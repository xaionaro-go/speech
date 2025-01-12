package goconv

import (
	"time"

	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

func TranscriptFromGRPC(t *speechtotext_grpc.Transcript) *speech.Transcript {
	return &speech.Transcript{
		Variants:        VariantsFromGRPC(t.GetVariants()),
		Stability:       t.GetStability(),
		AudioChannelNum: audio.Channel(t.GetAudioChannelNum()),
		Language:        speech.Language(t.GetLanguage()),
		IsFinal:         t.GetIsFinal(),
	}
}

func VariantsFromGRPC(variants []*speechtotext_grpc.TranscriptVariant) speech.TranscriptVariants {
	result := make(speech.TranscriptVariants, 0, len(variants))
	for _, variant := range variants {
		result = append(result, VariantFromGRPC(variant))
	}
	return result
}

func VariantFromGRPC(variant *speechtotext_grpc.TranscriptVariant) speech.TranscriptVariant {
	return speech.TranscriptVariant{
		Text:             speech.Text(variant.GetText()),
		TranscriptTokens: TranscriptTokensFromGRPC(variant.GetTranscriptTokens()),
		Confidence:       variant.GetConfidence(),
	}
}

func TranscriptTokensFromGRPC(tokens []*speechtotext_grpc.TranscriptToken) speech.TranscriptTokens {
	result := make(speech.TranscriptTokens, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, TranscriptTokenFromGRPC(token))
	}
	return result
}

func TranscriptTokenFromGRPC(variant *speechtotext_grpc.TranscriptToken) speech.TranscriptToken {
	return speech.TranscriptToken{
		StartTime:  time.Duration(variant.GetStartTimeNano()) * time.Nanosecond,
		EndTime:    time.Duration(variant.GetEndTimeNano()) * time.Nanosecond,
		Text:       speech.Text(variant.GetText()),
		Confidence: variant.GetConfidence(),
		Speaker:    variant.GetSpeaker(),
	}
}
