package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/lazybeaver/entropy"
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
	useGPUFlag := pflag.Bool("use-gpu", true, "")
	remoteFlag := pflag.String("remote-addr", "", "use a remote speech-to-text engine, instead of running it locally")
	shouldTranslateFlag := pflag.Bool("translate", false, "")
	printTimestampsFlag := pflag.Bool("print-timestamps", false, "")
	printTokenTimestampsFlag := pflag.Bool("print-token-timestamps", false, "")
	printConfidencesFlag := pflag.Bool("print-confidences", false, "")
	printEntropyFlag := pflag.Bool("print-entropy", false, "")
	printNoSpeechProbabilityFlag := pflag.Bool("print-no-speech-probability", false, "")
	netPprofAddr := pflag.String("net-pprof-listen-addr", "", "an address to listen for incoming net/pprof connections")
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

	if *netPprofAddr != "" {
		observability.Go(ctx, func() { l.Error(http.ListenAndServe(*netPprofAddr, nil)) })
	}

	var whisperModel []byte
	if *remoteFlag == "" || whisperModelPath != "" {
		var err error
		whisperModel, err = os.ReadFile(whisperModelPath)
		if err != nil {
			panic(err)
		}
	}

	var opts whisper.Options
	if *gpuFlag != -1 {
		opts = append(opts, whisper.OptionGPUDeviceID(*gpuFlag))
	}
	opts = append(opts, whisper.OptionUseGPU(*useGPUFlag))

	var (
		stt speech.ToText
		err error
	)
	if *remoteFlag != "" {
		logger.Debugf(ctx, "initializing a remote context")
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
		logger.Debugf(ctx, "initializing a local context")
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
					"%8s - %8s: %s",
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
			if *printTokenTimestampsFlag {
				var tss []string
				for _, token := range variant.TranscriptTokens {
					tss = append(tss, fmt.Sprintf("%s-%s", token.StartTime, token.EndTime))
				}
				text += fmt.Sprintf(" | %s", strings.Join(tss, ", "))
			}
			if *printEntropyFlag {
				entropy, err := entropy.Shannon(string(variant.Text))
				text += fmt.Sprintf(" | %f (%v)", entropy, err)
			}
			if *printNoSpeechProbabilityFlag {
				text += fmt.Sprintf(" | %f", t.NoSpeechProbability)
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
