package goconv

import (
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper/types"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

func SamplingStrategyFromGRPC(
	s speechtotext_grpc.WhisperSamplingStrategy,
) types.SamplingStrategy {
	switch s {
	case speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyUndefined:
		return types.SamplingStrategyUndefined
	case speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyGreedy:
		return types.SamplingStrategyGreedy
	case speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyBreamSearch:
		return types.SamplingStrategyBreamSearch
	}
	return types.SamplingStrategyUndefined
}

func SamplingStrategyToGRPC(
	s types.SamplingStrategy,
) speechtotext_grpc.WhisperSamplingStrategy {
	switch s {
	case types.SamplingStrategyUndefined:
		return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyUndefined
	case types.SamplingStrategyGreedy:
		return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyGreedy
	case types.SamplingStrategyBreamSearch:
		return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyBreamSearch
	}
	return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyUndefined
}
