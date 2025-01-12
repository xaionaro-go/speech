package whisper

import (
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mutablelogic/go-whisper/sys/whisper"
)

// #include <whisper.h>
// #cgo pkg-config: libwhisper
// #cgo linux pkg-config: libwhisper-linux
// #cgo darwin pkg-config: libwhisper-darwin
import "C"

func logLevelFromWhisper(logLevel whisper.LogLevel) logger.Level {
	switch logLevel {
	case C.GGML_LOG_LEVEL_NONE:
		return logger.LevelPanic
	case whisper.LogLevelDebug:
		return logger.LevelDebug
	case whisper.LogLevelInfo:
		return logger.LevelInfo
	case whisper.LogLevelWarn:
		return logger.LevelWarning
	case whisper.LogLevelError:
		return logger.LevelError
	case C.GGML_LOG_LEVEL_CONT:
		return logger.LevelInfo
	}
	return logger.LevelUndefined
}
