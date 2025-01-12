package goconv

import (
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

func TranscriptToGRPC(t *speech.Transcript) *speechtotext_grpc.Transcript {
	return &speechtotext_grpc.Transcript{
		Variants:        VariantsToGRPC(t.Variants),
		Stability:       t.Stability,
		AudioChannelNum: uint32(t.AudioChannelNum),
		Language:        string(t.Language),
		IsFinal:         t.IsFinal,
	}
}

func VariantsToGRPC(variants speech.TranscriptVariants) []*speechtotext_grpc.TranscriptVariant {
	result := make([]*speechtotext_grpc.TranscriptVariant, 0, len(variants))
	for _, variant := range variants {
		result = append(result, VariantToGRPC(variant))
	}
	return result
}

func VariantToGRPC(variant speech.TranscriptVariant) *speechtotext_grpc.TranscriptVariant {
	return &speechtotext_grpc.TranscriptVariant{
		Text:             string(variant.Text),
		TranscriptTokens: TranscriptTokensToGRPC(variant.TranscriptTokens),
		Confidence:       variant.Confidence,
	}
}

func TranscriptTokensToGRPC(tokens speech.TranscriptTokens) []*speechtotext_grpc.TranscriptToken {
	result := make([]*speechtotext_grpc.TranscriptToken, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, TranscriptTokenToGRPC(token))
	}
	return result
}

func TranscriptTokenToGRPC(variant speech.TranscriptToken) *speechtotext_grpc.TranscriptToken {
	return &speechtotext_grpc.TranscriptToken{
		StartTimeNano: variant.StartTime.Nanoseconds(),
		EndTimeNano:   variant.EndTime.Nanoseconds(),
		Text:          string(variant.Text),
		Confidence:    variant.Confidence,
		Speaker:       variant.Speaker,
	}
}
