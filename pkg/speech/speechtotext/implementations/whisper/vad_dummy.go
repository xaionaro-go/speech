//go:build no_libfvad
// +build no_libfvad

package whisper

import (
	"context"

	"github.com/xaionaro-go/speech/pkg/vad"
)

func (stt *SpeechToText) newVAD(
	_ context.Context,
) (vad.VAD, error) {
	return vad.NewDummy(stt.AudioEncodingNoErr(), stt.AudioChannelsNoErr()), nil
}
