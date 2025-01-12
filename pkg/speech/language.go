package speech

import "strings"

type Language string

const (
	LanguageEnglishUS = "en-US"
	LanguageRussian   = "ru-RU"
)

type LanguageFamily string

func (l Language) Family() LanguageFamily {
	words := strings.SplitN(string(l), "-", 2)
	return LanguageFamily(words[0])
}
