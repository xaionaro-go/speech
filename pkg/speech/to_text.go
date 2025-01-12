package speech

import (
	"context"
	"io"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type Text string

func (t Text) ContainsAlphaNum() bool {
	return strings.ContainsFunc(string(t), func(r rune) bool {
		if r == '-' {
			return false
		}
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	})
}

type TranscriptToken struct {
	StartTime  time.Duration
	EndTime    time.Duration
	Text       Text
	Confidence float32
	Speaker    string
}

func (t *TranscriptToken) ContainsAlphaNum() bool {
	return t.Text.ContainsAlphaNum()
}

type TranscriptTokens []TranscriptToken

func (s TranscriptTokens) StartTime() time.Duration {
	for _, token := range s {
		if !token.ContainsAlphaNum() {
			continue
		}
		if token.StartTime != token.EndTime {
			return token.StartTime
		}
	}
	return 0
}

func (s TranscriptTokens) EndTime() time.Duration {
	for _, token := range slices.Backward(s) {
		if !token.ContainsAlphaNum() {
			continue
		}
		if token.StartTime != token.EndTime {
			return token.EndTime
		}
	}
	return 0
}

type TranscriptVariant struct {
	Text             Text
	TranscriptTokens TranscriptTokens
	Confidence       float32
}

func (v *TranscriptVariant) StartTime() time.Duration {
	return v.TranscriptTokens.StartTime()
}

func (v *TranscriptVariant) EndTime() time.Duration {
	return v.TranscriptTokens.EndTime()
}

type Transcript struct {
	Variants        []TranscriptVariant
	Stability       float32
	AudioChannelNum audio.Channel
	Language        Language
	IsFinal         bool
}

type ToText interface {
	io.Closer
	AudioEncoding() audio.Encoding
	AudioChannels() audio.Channel
	WriteAudio(context.Context, []byte) error
	OutputChan() <-chan *Transcript
}
