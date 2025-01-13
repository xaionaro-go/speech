package subtitleswindow

import (
	"context"
	"fmt"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/hashicorp/go-multierror"
	syswhisper "github.com/mutablelogic/go-whisper/sys/whisper"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/client"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/goconv"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
	"github.com/xaionaro-go/xsync"
)

const (
	maxLines = 5
	timeout  = time.Second * 30
)

type subtitlePiece struct {
	TS      time.Time
	Text    string
	IsFinal bool
}

type speechRecognizer struct {
	ctx          context.Context
	cancelFunc   context.CancelFunc
	renderLocker xsync.Gorex
	window       *SubtitlesWindow
	audioInput   io.Reader
	whisper      speech.ToText
	subtitles    []subtitlePiece
	onceCloser   onceCloser
}

// audioInput is supposed to be PCM Float32LE 16000Hz 1ch
func newSpeechRecognizer(
	ctx context.Context,
	audioInput io.Reader,
	remoteAddrWhisper string,
	gpu int,
	whisperModel []byte,
	language speech.Language,
	shouldTranslate bool,
	window *SubtitlesWindow,
) (*speechRecognizer, error) {
	var (
		stt speech.ToText
		err error
	)
	if remoteAddrWhisper == "" {
		logger.Debugf(ctx, "initializing a local context")
		var opts whisper.Options
		if gpu >= 0 {
			opts = append(opts, whisper.OptionGPUDeviceID(gpu))
		}
		stt, err = whisper.New(
			ctx,
			whisperModel,
			language,
			whisper.SamplingStrategyBreamSearch,
			shouldTranslate,
			syswhisper.AlignmentAheadsPresetNone,
			opts...,
		)
	} else {
		logger.Debugf(ctx, "initializing a remote context")
		stt, err = client.New(ctx, remoteAddrWhisper, &speechtotext_grpc.NewContextRequest{
			ModelBytes:      whisperModel,
			Language:        string(language),
			ShouldTranslate: shouldTranslate,
			Backend: &speechtotext_grpc.NewContextRequest_Whisper{
				Whisper: &speechtotext_grpc.WhisperOptions{
					SamplingStrategy:      goconv.SamplingStrategyToGRPC(whisper.SamplingStrategyGreedy),
					AlignmentAheadsPreset: goconv.AlignmentAheadsPresetToGRPC(syswhisper.AlignmentAheadsPreset(syswhisper.AlignmentAheadsPresetNone)),
				},
			},
		})
	}
	if err != nil {
		return nil, fmt.Errorf("whisper.New: %w", err)
	}

	ctx, cancelFn := context.WithCancel(ctx)
	r := &speechRecognizer{
		ctx:        ctx,
		cancelFunc: cancelFn,
		window:     window,
		audioInput: audioInput,
		whisper:    stt,
	}
	observability.Go(ctx, func() {
		defer r.Close()
		err := r.transcriptLoop(ctx)
		if err != nil && err != context.Canceled {
			select {
			case <-ctx.Done():
			default:
				logger.Errorf(ctx, "transcript loop is closed: %v", err)
			}
		}
	})

	observability.Go(ctx, func() {
		defer r.Close()
		err := r.audioWriterLoop(ctx)
		if err != nil && err != context.Canceled {
			select {
			case <-ctx.Done():
			default:
				logger.Errorf(ctx, "audio writer loop is closed: %v", err)
			}
		}
	})
	return r, nil
}

func (r *speechRecognizer) audioWriterLoop(
	ctx context.Context,
) (_err error) {
	logger.Debugf(ctx, "audioWriterLoop()")
	defer func() { logger.Debugf(ctx, "/audioWriterLoop(): %v", _err) }()

	buf := make([]byte, 1024*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger.Debugf(ctx, "audioWriterLoop(): reading audio")
		n, err := r.audioInput.Read(buf)
		logger.Debugf(ctx, "/audioWriterLoop(): reading audio: %v %v", n, err)
		if err != nil {
			return fmt.Errorf("unable to read: %w", err)
		}
		if n == len(buf) {
			return fmt.Errorf("the message is too big: >=%d", len(buf))
		}

		msg := buf[:n]

		logger.Debugf(ctx, "audioWriterLoop(): writing audio: %d", len(msg))
		err = r.whisper.WriteAudio(ctx, msg)
		logger.Debugf(ctx, "/audioWriterLoop(): writing audio: %v", err)
		if err != nil {
			return fmt.Errorf("unable to write audio of length %d to whisper: %w", len(msg), err)
		}
	}
}

func (r *speechRecognizer) transcriptLoop(
	ctx context.Context,
) (_err error) {
	logger.Debugf(ctx, "transcriptLoop()")
	defer func() { logger.Debugf(ctx, "/transcriptLoop(): %v", _err) }()

	t := time.NewTicker(time.Second)
	defer t.Stop()
	ch, err := r.whisper.OutputChan(ctx)
	if err != nil {
		return fmt.Errorf("unable to get the output chan: %w", err)
	}
	for {
		select {
		case <-t.C:
			r.render(ctx)
		case transcript, ok := <-ch:
			if !ok {
				return fmt.Errorf("the whisper client is closed")
			}
			err := r.addTranscript(ctx, transcript)
			if err != nil {
				logger.Errorf(ctx, "unable to render the transcript: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *speechRecognizer) addTranscript(
	ctx context.Context,
	transcript *speech.Transcript,
) (_err error) {
	logger.Debugf(ctx, "addTranscript(ctx, %#+v)", *transcript)
	defer func() { logger.Debugf(ctx, "/addTranscript(ctx, %#+v): %v", *transcript, _err) }()

	if len(transcript.Variants) == 0 {
		return fmt.Errorf("no variants provided")
	}
	r.renderLocker.Do(ctx, func() {
		variant := transcript.Variants[0]
		text := variant.Text
		resultingPiece := subtitlePiece{
			TS:      time.Now(),
			Text:    string(text),
			IsFinal: transcript.IsFinal,
		}

		if len(r.subtitles) > 0 && !r.subtitles[len(r.subtitles)-1].IsFinal {
			r.subtitles[len(r.subtitles)-1] = resultingPiece
		} else {
			if len(r.subtitles) >= maxLines {
				r.subtitles = r.subtitles[1:]
			}
			r.subtitles = append(r.subtitles, resultingPiece)
		}

		r.render(ctx)
	})

	return nil
}

func (r *speechRecognizer) render(
	ctx context.Context,
) {
	logger.Debugf(ctx, "render(ctx)")
	defer func() { logger.Debugf(ctx, "/render(ctx)") }()

	r.renderLocker.Do(ctx, func() {
		var lines []widget.RichTextSegment
		for _, piece := range r.subtitles {
			if piece.TS.After(time.Now().Add(-timeout)) {
				var style widget.RichTextStyle
				if piece.IsFinal {
					style = widget.RichTextStyle{
						Alignment: fyne.TextAlignCenter,
						ColorName: theme.ColorNameForeground,
						Inline:    false,
						SizeName:  theme.SizeNameHeadingText,
						TextStyle: fyne.TextStyle{
							Bold:      false,
							Italic:    false,
							Monospace: false,
							Symbol:    false,
							TabWidth:  0,
							Underline: false,
						},
					}
				} else {
					style = widget.RichTextStyle{
						Alignment: fyne.TextAlignCenter,
						ColorName: theme.ColorNameHyperlink,
						Inline:    false,
						SizeName:  theme.SizeNameHeadingText,
						TextStyle: fyne.TextStyle{
							Bold:      false,
							Italic:    false,
							Monospace: false,
							Symbol:    false,
							TabWidth:  0,
							Underline: false,
						},
					}
				}
				logger.Debugf(ctx, "resultText[%d] = '%s'", len(lines), piece.Text)
				lines = append(lines, &widget.TextSegment{
					Style: style,
					Text:  piece.Text,
				})
			}
		}

		textObj := widget.NewRichText(lines...)
		textObj.Wrapping = fyne.TextWrapWord
		r.window.Container.RemoveAll()
		r.window.Container.Add(textObj)
		r.window.Container.Refresh()
	})
}

func (r *speechRecognizer) Close() error {
	var mErr *multierror.Error
	r.onceCloser.Do(func() {
		logger.Debugf(context.TODO(), "Close")
		r.cancelFunc()
		if err := r.whisper.Close(); err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("whisperClient.Close(): %w", err))
		}
		if err := r.window.Close(); err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("windowCloser.Close(): %w", err))
		}
	})
	return mErr.ErrorOrNil()
}
