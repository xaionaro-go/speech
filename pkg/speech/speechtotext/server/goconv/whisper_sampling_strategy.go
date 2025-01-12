package goconv

import (
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

func SamplingStrategyFromGRPC(
	s speechtotext_grpc.WhisperSamplingStrategy,
) whisper.SamplingStrategy {
	switch s {
	case speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyUndefined:
		return whisper.SamplingStrategyUndefined
	case speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyGreedy:
		return whisper.SamplingStrategyGreedy
	case speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyBreamSearch:
		return whisper.SamplingStrategyBreamSearch
	}
	return whisper.SamplingStrategyUndefined
}

func SamplingStrategyToGRPC(
	s whisper.SamplingStrategy,
) speechtotext_grpc.WhisperSamplingStrategy {
	switch s {
	case whisper.SamplingStrategyUndefined:
		return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyUndefined
	case whisper.SamplingStrategyGreedy:
		return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyGreedy
	case whisper.SamplingStrategyBreamSearch:
		return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyBreamSearch
	}
	return speechtotext_grpc.WhisperSamplingStrategy_WhisperSamplingStrategyUndefined
}
