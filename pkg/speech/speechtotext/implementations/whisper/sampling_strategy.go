package whisper

import (
	"fmt"

	"github.com/mutablelogic/go-whisper/sys/whisper"
)

type SamplingStrategy int

const (
	SamplingStrategyUndefined = SamplingStrategy(iota)
	SamplingStrategyGreedy
	SamplingStrategyBreamSearch
)

func (ss SamplingStrategy) ToWhisper() whisper.SamplingStrategy {
	switch ss {
	case SamplingStrategyGreedy:
		return whisper.SAMPLING_GREEDY
	case SamplingStrategyBreamSearch:
		return whisper.SAMPLING_BEAM_SEARCH
	}
	panic(fmt.Errorf("unknown sampling strategy: %d", ss))
}
