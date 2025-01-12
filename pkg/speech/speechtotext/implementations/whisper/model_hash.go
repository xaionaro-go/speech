package whisper

import "encoding/hex"

func hexMustDecodeString(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

var (
	ModelHashMedium  = hexMustDecodeString("fd9727b6e1217c2f614f9b698455c4ffd82463b4")
	ModelHashLargeV3 = hexMustDecodeString("ad82bf6a9043ceed055076d0fd39f5f186ff8062")
)
