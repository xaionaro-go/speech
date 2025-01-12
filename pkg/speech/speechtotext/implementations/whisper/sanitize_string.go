package whisper

import "strings"

func sanitizeString(in string) string {
	in = strings.ReplaceAll(in, "!", "")
	in = strings.ReplaceAll(in, "?", "")
	in = strings.ReplaceAll(in, ".", "")
	return strings.ToLower(strings.Trim(in, " "))
}
