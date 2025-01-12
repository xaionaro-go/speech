package subtitleswindow

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/hashicorp/go-multierror"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/audio/resampler"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/player/pkg/player/builtin"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/whisper"
	"github.com/xaionaro-go/xsync"
)

const (
	maxLines = 5
	timeout  = time.Second * 30
)

type subtitlePiece struct {
	TS   time.Time
	Text string
}

type speechRecognizer struct {
	ctx          context.Context
	cancelFunc   context.CancelFunc
	renderLocker xsync.Gorex
	window       *SubtitlesWindow
	whisper      *whisper.SpeechToText
	subtitles    []subtitlePiece
	onceCloser   onceCloser
}

var _ builtin.AudioRenderer = (*speechRecognizer)(nil)

func newSpeechRecognizer(
	ctx context.Context,
	whisperModel []byte,
	language speech.Language,
	shouldTranslate bool,
	window *SubtitlesWindow,
) (*speechRecognizer, error) {
	stt, err := whisper.New(
		ctx,
		whisperModel,
		language,
		whisper.SamplingStrategyBreamSearch,
		shouldTranslate,
	)
	if err != nil {
		return nil, fmt.Errorf("whisper.New: %w", err)
	}

	ctx, cancelFn := context.WithCancel(ctx)
	r := &speechRecognizer{
		ctx:        ctx,
		cancelFunc: cancelFn,
		window:     window,
		whisper:    stt,
	}
	observability.Go(ctx, func() {
		defer r.Close()
		err := r.loop(ctx)
		if err != nil && err != context.Canceled {
			select {
			case <-ctx.Done():
			default:
				logger.Errorf(ctx, "loop is closed: %v", err)
			}
		}
	})
	return r, nil
}

func (r *speechRecognizer) loop(
	ctx context.Context,
) (_err error) {
	logger.Debugf(ctx, "loop()")
	defer func() { logger.Debugf(ctx, "/loop(): %v", _err) }()

	t := time.NewTicker(time.Second)
	defer t.Stop()
	ch := r.whisper.OutputChan()
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
		text := transcript.Variants[0].Text

		if len(r.subtitles) >= maxLines {
			r.subtitles = r.subtitles[1:]
		}
		r.subtitles = append(r.subtitles, subtitlePiece{
			TS:   time.Now(),
			Text: text,
		})
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
		var lines []string
		for _, piece := range r.subtitles {
			if piece.TS.After(time.Now().Add(-timeout)) {
				lines = append(lines, piece.Text)
			}
		}

		resultText := "# " + strings.Join(lines, "\n# ")
		logger.Debugf(ctx, "resultText = '%s'", resultText)
		textObj := widget.NewRichTextFromMarkdown(resultText)
		textObj.Wrapping = fyne.TextWrapWord
		r.window.Container.RemoveAll()
		r.window.Container.Add(textObj)
		r.window.Container.Refresh()
	})
}

func (r *speechRecognizer) PlayPCM(
	sampleRate audio.SampleRate,
	channels audio.Channel,
	format audio.PCMFormat,
	bufferSize time.Duration,
	reader io.Reader,
) (audio.Stream, error) {
	ctx := context.TODO()
	logger.Debugf(ctx, "PlayPCM(%v, %v, %v, %v, reader)", sampleRate, channels, format, bufferSize)
	requiredEncoding := r.whisper.AudioEncoding()
	requiredPCMEncoding, ok := requiredEncoding.(audio.EncodingPCM)
	if !ok {
		return nil, fmt.Errorf("the transcriptor requires a non-PCM encoding: %#+v", requiredEncoding)
	}

	myFormat := resampler.Format{
		Channels:   channels,
		SampleRate: sampleRate,
		PCMFormat:  format,
	}
	requiredFormat := resampler.Format{
		Channels:   r.whisper.AudioChannels(),
		SampleRate: requiredPCMEncoding.SampleRate,
		PCMFormat:  requiredPCMEncoding.PCMFormat,
	}

	resampledReader, err := resampler.NewResampler(myFormat, reader, requiredFormat)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a resampler from %#+v to %#+v: %w", myFormat, requiredFormat, err)
	}

	return newSpeechStream(r.ctx, resampledReader, r.whisper, r), nil
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
