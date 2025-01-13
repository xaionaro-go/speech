package libfvad

import (
	"context"
	"fmt"
	"math"

	"github.com/josharian/fvad"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/speech/pkg/vad"
)

type VAD struct {
	*fvad.Detector
	SampleRate audio.SampleRate
}

var _ vad.VAD = (*VAD)(nil)

func NewVAD(
	sampleRate audio.SampleRate,
	sensitivityMode int,
) (*VAD, error) {
	detector := fvad.NewDetector()
	if err := detector.SetSampleRate(int(sampleRate)); err != nil {
		return nil, fmt.Errorf("unable to set the sample rate: %w", err)
	}
	if err := detector.SetMode(sensitivityMode); err != nil {
		return nil, fmt.Errorf("unable to set the sensitivity mode: %w", err)
	}
	return &VAD{
		SampleRate: sampleRate,
		Detector:   detector,
	}, nil
}

func (vad *VAD) Close() error {
	vad.Detector.Close()
	return nil
}

func (vad *VAD) Encoding(context.Context) (audio.Encoding, error) {
	return vad.EncodingNoErr(), nil
}

func (vad *VAD) EncodingNoErr() audio.EncodingPCM {
	return audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatS16LE,
		SampleRate: vad.SampleRate,
	}
}

func (vad *VAD) Channels(context.Context) (audio.Channel, error) {
	return vad.ChannelsNoErr(), nil
}

func (vad *VAD) ChannelsNoErr() audio.Channel {
	return 1
}

func (vad *VAD) VoiceProbability(ctx context.Context, samples []byte) (float64, error) {
	// see the description of (*fvad.Detector).Process
	minPortion := 2 * 80 * vad.SampleRate / 8000
	midPortion := minPortion * 2
	maxPortion := minPortion * 3

	for {
		var frame []byte
		switch {
		case len(samples) >= int(maxPortion):
			frame = samples[:maxPortion]
		case len(samples) >= int(midPortion):
			frame = samples[:midPortion]
		case len(samples) >= int(minPortion):
			frame = samples[:minPortion]
		default:
			return 0, nil
		}
		samples = samples[len(frame):]
		result, err := vad.Detector.Process(convertBytesToInt16Slice(frame))
		if err != nil {
			return math.NaN(), err
		}
		if result {
			return 1, nil
		}
	}
}

/*
// Process reports whether voice has been detected in buf.
// Only frames with a length of 10, 20 or 30 ms are supported.
// For example at 8 kHz, len(buf) must be either 80, 160 or 240.
*/
