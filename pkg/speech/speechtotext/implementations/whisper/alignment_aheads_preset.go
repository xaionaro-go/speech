package whisper

import (
	"fmt"
	"strings"

	"github.com/mutablelogic/go-whisper/sys/whisper"
)

type AlignmentAheadsPreset whisper.AlignmentAheadsPreset

// String just implements fmt.Stringer, flag.Value and pflag.Value.
func (p AlignmentAheadsPreset) String() string {
	switch p {
	case whisper.AlignmentAheadsPresetNone:
		return "none"
	case whisper.AlignmentAheadsPresetTinyEn:
		return "tiny_en"
	case whisper.AlignmentAheadsPresetTiny:
		return "tiny"
	case whisper.AlignmentAheadsPresetSmallEn:
		return "small_en"
	case whisper.AlignmentAheadsPresetSmall:
		return "small"
	case whisper.AlignmentAheadsPresetBaseEn:
		return "base_en"
	case whisper.AlignmentAheadsPresetBase:
		return "base"
	case whisper.AlignmentAheadsPresetMediumEn:
		return "medium_en"
	case whisper.AlignmentAheadsPresetMedium:
		return "medium"
	case whisper.AlignmentAheadsPresetLargeV1:
		return "large_v1"
	case whisper.AlignmentAheadsPresetLargeV2:
		return "large_v2"
	case whisper.AlignmentAheadsPresetLargeV3:
		return "large_v3"
	case whisper.AlignmentAheadsPresetNTopMost:
		return "n_top_most"
	case whisper.AlignmentAheadsPresetCustom:
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
		return whisper.AlignmentAheadsPresetNone, nil
	case "tiny_en":
		return whisper.AlignmentAheadsPresetTinyEn, nil
	case "tiny":
		return whisper.AlignmentAheadsPresetTiny, nil
	case "small_en":
		return whisper.AlignmentAheadsPresetSmallEn, nil
	case "small":
		return whisper.AlignmentAheadsPresetSmall, nil
	case "base_en":
		return whisper.AlignmentAheadsPresetBaseEn, nil
	case "base":
		return whisper.AlignmentAheadsPresetBase, nil
	case "medium_en":
		return whisper.AlignmentAheadsPresetMediumEn, nil
	case "medium":
		return whisper.AlignmentAheadsPresetMedium, nil
	case "large_v1":
		return whisper.AlignmentAheadsPresetLargeV1, nil
	case "large_v2":
		return whisper.AlignmentAheadsPresetLargeV2, nil
	case "large_v3":
		return whisper.AlignmentAheadsPresetLargeV3, nil
	case "n_top_most":
		return whisper.AlignmentAheadsPresetNTopMost, nil
	case "custom":
		return whisper.AlignmentAheadsPresetCustom, nil
	}
	var allowedValues []string
	for aap := AlignmentAheadsPreset(0); aap <= whisper.AlignmentAheadsPresetLargeV3; aap++ {
		allowedValues = append(allowedValues, aap.String())
	}
	return whisper.AlignmentAheadsPresetNone, fmt.Errorf("unknown alignment aheads preset '%s', known values are: %s",
		in, strings.Join(allowedValues, ", "))
}
