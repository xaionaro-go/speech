//go:build !no_libfvad
// +build !no_libfvad

package whisper

import (
	"context"

	"github.com/xaionaro-go/speech/pkg/vad"
	"github.com/xaionaro-go/speech/pkg/vad/implementations/libfvad"
)

func (stt *SpeechToText) newVAD(
	_ context.Context,
) (vad.VAD, error) {
	return libfvad.NewVAD(16000, 3)
}
