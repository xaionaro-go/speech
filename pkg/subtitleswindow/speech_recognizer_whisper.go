//go:build !no_whisper
// +build !no_whisper

package subtitleswindow

import (
	"context"

	syswhisper "github.com/mutablelogic/go-whisper/sys/whisper"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper/types"
)

func initLocalSTT(
	ctx context.Context,
	gpu int,
	whisperModel []byte,
	language speech.Language,
	shouldTranslate bool,
	vadThreshold float64,
) (speech.ToText, error) {
	var opts whisper.Options
	if gpu >= 0 {
		opts = append(opts, whisper.OptionGPUDeviceID(gpu))
	}
	return whisper.New(
		ctx,
		whisperModel,
		language,
		types.SamplingStrategyBreamSearch,
		shouldTranslate,
		syswhisper.AlignmentAheadsPresetNone,
		vadThreshold,
		opts...,
	)
}
