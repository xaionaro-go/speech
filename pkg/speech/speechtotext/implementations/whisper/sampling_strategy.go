package whisper

import (
	"fmt"

	"github.com/mutablelogic/go-whisper/sys/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper/types"
)

type SamplingStrategy types.SamplingStrategy

func (ss SamplingStrategy) ToWhisper() whisper.SamplingStrategy {
	switch types.SamplingStrategy(ss) {
	case types.SamplingStrategyGreedy:
		return whisper.SAMPLING_GREEDY
	case types.SamplingStrategyBreamSearch:
		return whisper.SAMPLING_BEAM_SEARCH
	}
	panic(fmt.Errorf("unknown sampling strategy: %d", ss))
}
