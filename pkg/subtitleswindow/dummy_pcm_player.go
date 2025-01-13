package subtitleswindow

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/audio/resampler"
	"github.com/xaionaro-go/player/pkg/player/builtin"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
)

type dummyPCMPlayer struct {
	*io.PipeReader
	ctx    context.Context
	writer *io.PipeWriter
}

var _ io.Reader = (*dummyPCMPlayer)(nil)
var _ builtin.AudioRenderer = (*dummyPCMPlayer)(nil)

func NewDummyPCMPlayer(ctx context.Context) *dummyPCMPlayer {
	r, w := io.Pipe()
	return &dummyPCMPlayer{
		PipeReader: r,
		ctx:        ctx,
		writer:     w,
	}
}

func (r *dummyPCMPlayer) PlayPCM(
	sampleRate audio.SampleRate,
	channels audio.Channel,
	format audio.PCMFormat,
	bufferSize time.Duration,
	reader io.Reader,
) (audio.PlayStream, error) {
	ctx := context.TODO()
	logger.Debugf(ctx, "PlayPCM(%v, %v, %v, %v, reader)", sampleRate, channels, format, bufferSize)
	requiredPCMEncoding := (*whisper.SpeechToText)(nil).AudioEncodingNoErr()

	myFormat := resampler.Format{
		Channels:   channels,
		SampleRate: sampleRate,
		PCMFormat:  format,
	}
	requiredFormat := resampler.Format{
		Channels:   (*whisper.SpeechToText)(nil).AudioChannelsNoErr(),
		SampleRate: requiredPCMEncoding.SampleRate,
		PCMFormat:  requiredPCMEncoding.PCMFormat,
	}

	resampledReader, err := resampler.NewResampler(myFormat, reader, requiredFormat)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a resampler from %#+v to %#+v: %w", myFormat, requiredFormat, err)
	}

	return newAudioStreamCopier(r.ctx, resampledReader, r.writer, r), nil
}

func (r *dummyPCMPlayer) Close() error {
	r.PipeReader.Close()
	r.writer.Close()
	return nil
}
