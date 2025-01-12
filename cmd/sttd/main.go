package main

import (
	"context"
	"fmt"
	"os"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server"
)

func syntaxExit(message string) {
	fmt.Fprintf(os.Stderr, "syntax error: %s\n", message)
	pflag.Usage()
	os.Exit(2)
}

func main() {
	loggerLevel := logger.LevelWarning
	pflag.Var(&loggerLevel, "log-level", "Log level")
	gpuFlag := pflag.Int("gpu", -1, "")
	useGPUFlag := pflag.Bool("use-gpu", true, "")
	contextsFlag := pflag.Uint("contexts", 1, "")
	pflag.Parse()
	if pflag.NArg() != 1 {
		syntaxExit("expected one argument (bind address)")
	}
	listenAddr := pflag.Arg(0)

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	listener, err := getListener(ctx, listenAddr)
	if err != nil {
		logger.Fatal(ctx, err)
	}

	var opts whisper.Options
	if *gpuFlag != -1 {
		opts = append(opts, whisper.OptionGPUDeviceID(*gpuFlag))
	}
	opts = append(opts, whisper.OptionUseGPU(*useGPUFlag))

	srv := server.NewServer(*contextsFlag, server.OptionWhisperOptions(opts))

	logger.Infof(ctx, "started at %v", listener.Addr())
	err = srv.Serve(ctx, listener)
	logger.Fatal(ctx, err)
}
