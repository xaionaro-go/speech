//go:build no_libfvad || windows
// +build no_libfvad windows

package whisper

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/vad"
)

func (stt *SpeechToText) newVAD(
	_ context.Context,
) (vad.VAD, error) {
	return vad.NewDummy(stt.AudioEncodingNoErr(), stt.AudioChannelsNoErr()), nil
}
