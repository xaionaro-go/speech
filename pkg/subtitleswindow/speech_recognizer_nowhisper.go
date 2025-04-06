//go:build no_whisper
// +build no_whisper

package subtitleswindow

import (
	"context"
	"fmt"

	"github.com/xaionaro-go/speech/pkg/speech"
)

func initLocalSTT(
	ctx context.Context,
	gpu int,
	whisperModel []byte,
	language speech.Language,
	shouldTranslate bool,
	vadThreshold float64,
) (speech.ToText, error) {
	return nil, fmt.Errorf("built without whisper")
}
