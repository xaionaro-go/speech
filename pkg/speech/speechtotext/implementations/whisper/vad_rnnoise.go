//go:build rnnoise && !windows

package whisper

import (
	"context"
	"fmt"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/noisesuppression/implementations/rnnoise"
	"github.com/xaionaro-go/audio/pkg/vad"
	"github.com/xaionaro-go/audio/pkg/vad/implementations/noisesuppression"
)

const (
	VADMinVoiceDuration = time.Nanosecond
	VADKeepContext      = 0
)

func (stt *SpeechToText) newVAD(
	ctx context.Context,
) (vad.VAD, error) {
	logger.Debugf(ctx, "newVAD:rnnoise")
	ns, err := rnnoise.New(1)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a RNNoise: %w", err)
	}
	vad, err := noisesuppression.NewVAD(ctx, ns, 10*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize VAD: %w", err)
	}
	return vad, nil
}
