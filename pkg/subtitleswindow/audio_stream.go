package subtitleswindow

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/observability"
)

type audioStreamCopier struct {
	cancelFunc   context.CancelFunc
	audioReader  io.Reader
	audioWriter  io.Writer
	playerCloser io.Closer
	wg           sync.WaitGroup
	onceCloser   onceCloser
}

var _ audio.Stream = (*audioStreamCopier)(nil)

func newAudioStreamCopier(
	ctx context.Context,
	audioReader io.Reader,
	audioWriter io.Writer,
	playerCloser io.Closer,
) *audioStreamCopier {
	ctx, cancelFunc := context.WithCancel(ctx)
	s := &audioStreamCopier{
		cancelFunc:   cancelFunc,
		audioReader:  audioReader,
		audioWriter:  audioWriter,
		playerCloser: playerCloser,
	}
	s.init(ctx)
	return s
}

func (s *audioStreamCopier) init(ctx context.Context) {
	s.wg.Add(1)
	observability.Go(ctx, func() {
		defer s.wg.Done()
		defer s.Close()
		err := s.loop(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				logger.Errorf(ctx, "loop returned error: %v", err)
			}
		}
	})
}
func (s *audioStreamCopier) loop(ctx context.Context) (_err error) {
	logger.Debugf(ctx, "loop()")
	defer func() { logger.Debugf(ctx, "/loop(): %v", _err) }()

	buf := make([]byte, 1024*1024)
	for {
		logger.Tracef(ctx, "Read()")
		n, err := s.audioReader.Read(buf)
		logger.Tracef(ctx, "/Read(): %v %v", n, err)
		if err != nil {
			return fmt.Errorf("unable to read audio from the reader: %w", err)
		}
		if n == len(buf) {
			return fmt.Errorf("message is too long; not implemented yet")
		}
		msg := buf[:n]

		logger.Tracef(ctx, "WriteAudio()")
		n, err = s.audioWriter.Write(msg)
		logger.Tracef(ctx, "/WriteAudio(): %v", err)
		if err != nil {
			return fmt.Errorf("unable to write the audio to the whisper: %w", err)
		}
		if n != len(msg) {
			return fmt.Errorf("written message is of invalid size: %d != %d", n, len(msg))
		}
	}
}

func (s *audioStreamCopier) Drain() error {
	return nil
}
func (s *audioStreamCopier) Close() error {
	var err error
	s.onceCloser.Do(func() {
		logger.Debugf(context.TODO(), "Close")
		s.cancelFunc()
		err = s.playerCloser.Close()
	})
	return err
}
