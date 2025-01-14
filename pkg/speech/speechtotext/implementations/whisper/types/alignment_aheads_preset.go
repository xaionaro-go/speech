package types

import (
	"fmt"
	"strings"

	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

type AlignmentAheadsPreset speechtotext_grpc.WhisperAlignmentAheadsPreset

// String just implements fmt.Stringer, flag.Value and pflag.Value.
func (p AlignmentAheadsPreset) String() string {
	switch p {
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetNone):
		return "none"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetTinyEn):
		return "tiny_en"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetTiny):
		return "tiny"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetSmallEn):
		return "small_en"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetSmall):
		return "small"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetBaseEn):
		return "base_en"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetBase):
		return "base"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetMediumEn):
		return "medium_en"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetMedium):
		return "medium"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV1):
		return "large_v1"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV2):
		return "large_v2"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV3):
		return "large_v3"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetNTopMost):
		return "n_top_most"
	case AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetCustom):
		return "custom"
	}
	return fmt.Sprintf("unknown_%d", p)
}

// Set updates the logging level values based on the passed string value.
// This method just implements flag.Value and pflag.Value.
func (p *AlignmentAheadsPreset) Set(value string) error {
	newLogLevel, err := ParseAlignmentAheadsPreset(value)
	if err != nil {
		return err
	}
	*p = newLogLevel
	return nil
}

// Type just implements pflag.Value.
func (p *AlignmentAheadsPreset) Type() string {
	return "AlignmentAheadsPreset"
}

func ParseAlignmentAheadsPreset(in string) (AlignmentAheadsPreset, error) {
	switch strings.ToLower(in) {
	case "none":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetNone), nil
	case "tiny_en":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetTinyEn), nil
	case "tiny":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetTiny), nil
	case "small_en":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetSmallEn), nil
	case "small":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetSmall), nil
	case "base_en":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetBaseEn), nil
	case "base":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetBase), nil
	case "medium_en":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetMediumEn), nil
	case "medium":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetMedium), nil
	case "large_v1":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV1), nil
	case "large_v2":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV2), nil
	case "large_v3":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV3), nil
	case "n_top_most":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetNTopMost), nil
	case "custom":
		return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetCustom), nil
	}
	var allowedValues []string
	for aap := AlignmentAheadsPreset(0); aap <= AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetLargeV3); aap++ {
		allowedValues = append(allowedValues, aap.String())
	}
	return AlignmentAheadsPreset(speechtotext_grpc.WhisperAlignmentAheadsPreset_WhisperAlignmentAheadsPresetNone), fmt.Errorf("unknown alignment aheads preset '%s', known values are: %s",
		in, strings.Join(allowedValues, ", "))
}
