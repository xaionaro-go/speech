package server

import (
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
)

type config struct {
	WhisperOptions whisper.Options
}

type Option interface {
	apply(*config)
}

type Options []Option

func (opts Options) apply(cfg *config) {
	for _, opt := range opts {
		opt.apply(cfg)
	}
}

func (opts Options) config() config {
	cfg := config{}
	opts.apply(&cfg)
	return cfg
}

type OptionWhisperOptions whisper.Options

func (opt OptionWhisperOptions) apply(cfg *config) {
	cfg.WhisperOptions = whisper.Options(opt)
}
