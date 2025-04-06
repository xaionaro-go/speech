//go:build (no_libfvad && !rnnoise) || windows

package whisper

import (
	"context"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/vad"
)

const (
	VADMinVoiceDuration = time.Nanosecond
	VADKeepContext      = 0
)

func (stt *SpeechToText) newVAD(
	ctx context.Context,
) (vad.VAD, error) {
	logger.Debugf(ctx, "newVAD:dummy")
	return vad.NewDummy(stt.AudioEncodingNoErr(), stt.AudioChannelsNoErr()), nil
}
