package subtitleswindow

import (
	"context"
	"fmt"
	"io"
	"sync"

	"fyne.io/fyne/v2"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/hashicorp/go-multierror"
	"github.com/xaionaro-go/speech/pkg/speech"
)

type SubtitlesWindow struct {
	fyne.Window
	speechRecognizer *speechRecognizer
	wg               sync.WaitGroup
	onceCloser       onceCloser
}

// audioInput is supposed to be PCM Float32LE 16000Hz 1ch
func New(
	ctx context.Context,
	app fyne.App,
	title string,
	textAlignment fyne.TextAlign,
	audioInput io.Reader,
	remoteAddrWhisper string,
	gpu int,
	whisperModel []byte,
	language speech.Language,
	shouldTranslate bool,
	translateOnlyFrom []speech.Language,
	vadThreshold float64,
) (_ret *SubtitlesWindow, _err error) {
	logger.Debugf(ctx, "New(ctx, app, '%s', audioInput, len:%d, translateOnlyFrom:%v)", title, len(whisperModel), translateOnlyFrom)
	defer func() {
		logger.Debugf(ctx, "/New(ctx, app, '%s', audioInput, len:%d, translateOnlyFrom:%v): %#+v %#+v", title, len(whisperModel), translateOnlyFrom, _ret, _err)
	}()

	w := &SubtitlesWindow{}

	w.Window = app.NewWindow(title)
	w.Window.Resize(fyne.NewSize(960, 600))

	var err error
	w.speechRecognizer, err = newSpeechRecognizer(ctx, textAlignment, audioInput, remoteAddrWhisper, gpu, whisperModel, language, shouldTranslate, translateOnlyFrom, vadThreshold, w)
	logger.Debugf(ctx, "newSpeechRecognizer(): %#+v %#+v", w.speechRecognizer, err)
	if err != nil {
		w.Window.Close()
		return nil, fmt.Errorf("unable to initialize a new speech recognizer: %w", err)
	}

	return w, nil
}

func (w *SubtitlesWindow) Wait() error {
	w.wg.Wait()
	return nil
}

func (w *SubtitlesWindow) Close() error {
	var mErr *multierror.Error
	w.onceCloser.Do(func() {
		ctx := context.TODO()
		logger.Debugf(ctx, "Close")

		if err := w.speechRecognizer.Close(); err != nil {
			mErr = multierror.Append(fmt.Errorf("unable to close the speech recognizer: %v", err))
		}

		w.Window.Close()
	})
	return mErr.ErrorOrNil()
}
