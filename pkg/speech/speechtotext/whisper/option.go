package whisper

type config struct {
	UseGPU      *bool
	GPUDeviceID *int
	FlashAttn   *bool
}

func defaultConfig() config {
	return config{}
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
	cfg := defaultConfig()
	opts.apply(&cfg)
	return cfg
}

type OptionUseGPU bool

func (opt OptionUseGPU) apply(cfg *config) {
	cfg.UseGPU = (*bool)(&opt)
}

type OptionGPUDeviceID int

func (opt OptionGPUDeviceID) apply(cfg *config) {
	cfg.GPUDeviceID = (*int)(&opt)
}

type OptionFlashAttn bool

func (opt OptionFlashAttn) apply(cfg *config) {
	cfg.FlashAttn = (*bool)(&opt)
}
