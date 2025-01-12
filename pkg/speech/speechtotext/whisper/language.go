package whisper

import (
	"github.com/xaionaro-go/speech/pkg/speech"
)

func LanguageToWhisper(language speech.Language) string {
	if language == "" {
		return "auto"
	}
	return string(language.Family())
}
