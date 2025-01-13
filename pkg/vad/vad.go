package vad

import (
	"context"
	"io"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type VAD interface {
	io.Closer

	Encoding(context.Context) (audio.Encoding, error)
	Channels(context.Context) (audio.Channel, error)

	VoiceProbability(context.Context, []byte) (float64, error)
}
