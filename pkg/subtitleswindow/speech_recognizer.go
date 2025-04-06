package subtitleswindow

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/hashicorp/go-multierror"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/client"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper/types"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/goconv"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
	"github.com/xaionaro-go/xsync"
)

const (
	maxLines = 5
	timeout  = time.Second * 30
)

type subtitlePiece struct {
	TS       time.Time
	Language speech.Language
	Text     string
	IsFinal  bool
}

type speechRecognizer struct {
	ctx               context.Context
	textAlignment     fyne.TextAlign
	cancelFunc        context.CancelFunc
	renderLocker      xsync.Gorex
	window            *SubtitlesWindow
	audioInput        io.Reader
	whisper           speech.ToText
	subtitles         []subtitlePiece
	shouldTranslate   bool
	translateOnlyFrom []speech.Language
	onceCloser        onceCloser
}

// audioInput is supposed to be PCM Float32LE 16000Hz 1ch
func newSpeechRecognizer(
	ctx context.Context,
	textAlignment fyne.TextAlign,
	audioInput io.Reader,
	remoteAddrWhisper string,
	gpu int,
	whisperModel []byte,
	language speech.Language,
	shouldTranslate bool,
	translateOnlyFrom []speech.Language,
	vadThreshold float64,
	window *SubtitlesWindow,
) (*speechRecognizer, error) {
	var (
		stt speech.ToText
		err error
	)
	if remoteAddrWhisper == "" {
		logger.Debugf(ctx, "initializing a local context")
		stt, err = initLocalSTT(
			ctx,
			gpu,
			whisperModel,
			language,
			shouldTranslate,
			vadThreshold,
		)
	} else {
		logger.Debugf(ctx, "initializing a remote context")
		stt, err = client.New(ctx, remoteAddrWhisper, &speechtotext_grpc.NewContextRequest{
			ModelBytes:      whisperModel,
			Language:        string(language),
			ShouldTranslate: shouldTranslate,
			VadThreshold:    float32(vadThreshold),
			Backend: &speechtotext_grpc.NewContextRequest_Whisper{
				Whisper: &speechtotext_grpc.WhisperOptions{
					SamplingStrategy:      goconv.SamplingStrategyToGRPC(types.SamplingStrategyGreedy),
					AlignmentAheadsPreset: speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetNone,
				},
			},
		})
	}
	if err != nil {
		return nil, fmt.Errorf("whisper.New: %w", err)
	}

	ctx, cancelFn := context.WithCancel(ctx)
	r := &speechRecognizer{
		ctx:               ctx,
		textAlignment:     textAlignment,
		cancelFunc:        cancelFn,
		window:            window,
		audioInput:        audioInput,
		whisper:           stt,
		shouldTranslate:   shouldTranslate,
		translateOnlyFrom: translateOnlyFrom,
	}
	observability.Go(ctx, func() {
		defer r.Close()
		err := r.transcriptLoop(ctx, language)
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
	language speech.Language,
) (_err error) {
	logger.Debugf(ctx, "transcriptLoop('%s')", language)
	defer func() { logger.Debugf(ctx, "/transcriptLoop('%s'): %v", language, _err) }()

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

			logger.Debugf(ctx, "r.shouldTranslate == %t", r.shouldTranslate)
			if r.shouldTranslate {
				if r.translateOnlyFrom != nil {
					found := false
					for _, lang := range r.translateOnlyFrom {
						logger.Debugf(
							ctx,
							"transcript.Language.Family() == language.Family(): %v == %v: %t",
							transcript.Language.Family(), language.Family(),
							transcript.Language.Family() == language.Family(),
						)
						if transcript.Language.Family() == lang.Family() {
							found = true
							break
						}
					}
					if !found {
						logger.Debugf(ctx, "we are expected to print only translations from %v, but this is '%s'", r.translateOnlyFrom, transcript.Language)
						continue
					}
				} else {
					logger.Debugf(
						ctx,
						"transcript.Language.Family() == language.Family(): %v == %v: %t",
						transcript.Language.Family(), language.Family(),
						transcript.Language.Family() == language.Family(),
					)
					if transcript.Language.Family() == language.Family() {
						logger.Debugf(ctx, "we are expected to print only translations, but this is already '%s'", transcript.Language)
						continue
					}
				}
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
			TS:       time.Now(),
			Language: transcript.Language,
			Text:     string(text),
			IsFinal:  transcript.IsFinal,
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
		var cObjs []fyne.CanvasObject
		for _, piece := range r.subtitles {
			if !piece.TS.After(time.Now().Add(-timeout)) {
				continue
			}
			logger.Debugf(ctx, "resultText[%d] = '%s'", len(cObjs), piece.Text)
			cObjs = append(cObjs, r.generateLine(ctx, piece.Language, piece.Text, piece.IsFinal)...)
		}
		if r.shouldTranslate && len(cObjs) > 0 {
			cObjs = append(r.generateLine(ctx, "", "Auto-translation:", true), cObjs...)
		}

		textContainer := container.NewVBox(cObjs...)
		textContainer.Resize(r.window.Canvas().Size())
		background := canvas.NewRectangle(color.Gray{Y: 0})
		r.window.SetContent(
			container.NewStack(
				background,
				textContainer,
			),
		)
	})
}

func (r *speechRecognizer) generateLine(
	ctx context.Context,
	language speech.Language,
	text string,
	isFinal bool,
) []fyne.CanvasObject {
	foregroundColor := color.RGBA{255, 255, 255, 255}
	if !isFinal {
		foregroundColor = color.RGBA{255, 0, 128, 255}
	}

	langFamily := language.Family()
	if len(langFamily) > 0 {
		text = fmt.Sprintf("%s [%s]", text, langFamily)
	}

	const fontSize = 32

	var layouts []fyne.CanvasObject
	for text != "" {
		var objs []fyne.CanvasObject
		nextLineText := ""
		fgText := canvas.NewText(text, foregroundColor)
		for {
			fgText.TextSize = fontSize
			if fgText.MinSize().Width <= r.window.Canvas().Size().Width-20 {
				break
			}
			nextLineText = fgText.Text[len(fgText.Text)-1:len(fgText.Text)] + nextLineText
			fgText.Text = fgText.Text[:len(fgText.Text)-1]
		}
		logger.Tracef(ctx, "text = '%s', nextLineText = '%s'", fgText.Text, nextLineText)
		background := canvas.NewRectangle(color.RGBA{255, 0, 128, 64})
		background.Resize(fgText.MinSize())

		objs = append(objs, background)
		objs = append(objs, fgText)
		layout := container.NewWithoutLayout(objs...)
		layouts = append(layouts, layout)
		text = nextLineText
	}
	return layouts
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
