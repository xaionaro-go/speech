package goconv

import (
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper/types"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

func AlignmentAheadsPresetFromGRPC(
	p speechtotext_grpc.WhisperAlignmentAheadsPreset,
) types.AlignmentAheadsPreset {
	return types.AlignmentAheadsPreset(p)
}

func AlignmentAheadsPresetToGRPC(
	p types.AlignmentAheadsPreset,
) speechtotext_grpc.WhisperAlignmentAheadsPreset {
	return speechtotext_grpc.WhisperAlignmentAheadsPreset(p)
}
