package consts

import (
	"github.com/xaionaro-go/audio/pkg/audio"
)

const (
	AudioChannels = audio.Channel(1)
)

func AudioEncoding() audio.EncodingPCM {
	return audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat32LE,
		SampleRate: 16000,
	}
}
