package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	syswhisper "github.com/mutablelogic/go-whisper/sys/whisper"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/client"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/goconv"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
)

func syntaxExit(message string) {
	fmt.Fprintf(os.Stderr, "syntax error: %s\n", message)
	pflag.Usage()
	os.Exit(2)
}

func main() {
	loggerLevel := logger.LevelWarning
	pflag.Var(&loggerLevel, "log-level", "Log level")
	langFlag := pflag.String("language", "en-US", "")
	alignmentAheadPresentFlag := whisper.AlignmentAheadsPreset(syswhisper.AlignmentAheadsPresetNone)
	pflag.Var(&alignmentAheadPresentFlag, "alignment-aheads-preset", "")
	gpuFlag := pflag.Int("gpu", -1, "")
	remoteFlag := pflag.String("remote-addr", "", "use a remote speech-to-text engine, instead of running it locally")
	shouldTranslateFlag := pflag.Bool("translate", false, "")
	printTimestampsFlag := pflag.Bool("print-timestamps", false, "")
	printConfidencesFlag := pflag.Bool("print-confidences", false, "")
	pflag.Parse()
	if pflag.NArg() != 1 {
		syntaxExit("expected one argument (whisper model path)")
	}

	whisperModelPath := pflag.Arg(0)

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	whisperModel, err := os.ReadFile(whisperModelPath)
	if err != nil {
		logger.Fatal(ctx, err)
	}

	var opts whisper.Options
	if *gpuFlag != -1 {
		opts = append(opts, whisper.OptionGPUDeviceID(*gpuFlag))
	}

	var stt speech.ToText
	if *remoteFlag != "" {
		stt, err = client.New(ctx, *remoteFlag, &speechtotext_grpc.NewContextRequest{
			ModelBytes:      whisperModel,
			Language:        *langFlag,
			ShouldTranslate: *shouldTranslateFlag,
			Backend: &speechtotext_grpc.NewContextRequest_Whisper{
				Whisper: &speechtotext_grpc.WhisperOptions{
					SamplingStrategy:      goconv.SamplingStrategyToGRPC(whisper.SamplingStrategyGreedy),
					AlignmentAheadsPreset: goconv.AlignmentAheadsPresetToGRPC(syswhisper.AlignmentAheadsPreset(alignmentAheadPresentFlag)),
				},
			},
		})
	} else {
		stt, err = whisper.New(
			ctx,
			whisperModel,
			speech.Language(*langFlag),
			whisper.SamplingStrategyGreedy,
			*shouldTranslateFlag,
			syswhisper.AlignmentAheadsPreset(alignmentAheadPresentFlag),
			opts...,
		)
	}
	if err != nil {
		logger.Fatal(ctx, err)
	}
	defer stt.Close()
	logger.Infof(ctx, "initialized a Speech-To-Text engine")

	ch, err := stt.OutputChan(ctx)
	if err != nil {
		logger.Fatal(ctx, err)
	}

	observability.Go(ctx, func() {
		defer logger.Infof(ctx, "stopped reader")
		logger.Infof(ctx, "started reader")
		previousMessageLength := 0
		for t := range ch {
			variant := t.Variants[0]
			fmt.Printf("\r%s", strings.Repeat(" ", previousMessageLength))
			text := strings.ReplaceAll(string(variant.Text), "\n", "|")
			if *printTimestampsFlag {
				text = fmt.Sprintf(
					"%8s - %8v: %s",
					variant.StartTime().Truncate(100*time.Millisecond),
					variant.EndTime().Truncate(100*time.Millisecond),
					text,
				)
			}
			if *printConfidencesFlag {
				var probs []string
				for _, token := range variant.TranscriptTokens {
					probs = append(probs, fmt.Sprintf("%f", token.Confidence))
				}
				text += fmt.Sprintf(" | %s", strings.Join(probs, ", "))
			}
			fmt.Printf("\r%s", text)
			previousMessageLength = len(text)
			if t.IsFinal {
				fmt.Printf("\n")
			}
		}
	})

	defer logger.Infof(ctx, "stopped writer")
	logger.Infof(ctx, "started writer")
	buf := make([]byte, 1024*1024)
	for {
		n, err := os.Stdin.Read(buf)
		if n == 0 && err != nil {
			break
		}
		err = stt.WriteAudio(ctx, buf[:n])
		if err != nil {
			logger.Fatal(ctx, err)
		}
	}
}
