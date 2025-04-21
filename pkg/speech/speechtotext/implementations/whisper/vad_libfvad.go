//go:build !no_libfvad && !rnnoise && !windows

package whisper

import (
	"context"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/vad"
	"github.com/xaionaro-go/audio/pkg/vad/implementations/libfvad"
)

const (
	VADMinVoiceDuration = 150 * time.Millisecond
	VADKeepContext      = time.Second
)

func (stt *SpeechToText) newVAD(
	ctx context.Context,
) (vad.VAD, error) {
	logger.Debugf(ctx, "newVAD:libfvad")
	return libfvad.NewVAD(16000, 3)
}
