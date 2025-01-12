package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"fyne.io/fyne/v2/app"
	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/subtitleswindow"
)

func syntaxExit(message string) {
	fmt.Fprintf(os.Stderr, "syntax error: %s\n", message)
	pflag.Usage()
	os.Exit(2)
}

func main() {
	loggerLevel := logger.LevelDebug
	pflag.Var(&loggerLevel, "log-level", "Log level")
	langFlag := pflag.String("language", "en-US", "")
	shouldTranslateFlag := pflag.Bool("translate", false, "")
	netPprofAddr := pflag.String("net-pprof-listen-addr", "", "an address to listen for incoming net/pprof connections")
	pflag.Parse()
	if pflag.NArg() != 2 {
		syntaxExit("expected two arguments: media-URL whisper-model-path")
	}

	mediaURL := pflag.Arg(0)
	whisperModelPath := pflag.Arg(1)

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	if *netPprofAddr != "" {
		observability.Go(ctx, func() { l.Error(http.ListenAndServe(*netPprofAddr, nil)) })
	}

	whisperModel, err := os.ReadFile(whisperModelPath)
	if err != nil {
		panic(err)
	}

	app := app.New()
	w, err := subtitleswindow.New(ctx, app, "Subtitles", mediaURL, whisperModel, speech.Language(*langFlag), *shouldTranslateFlag)
	if err != nil {
		panic(err)
	}
	w.Show()
	logger.Debugf(ctx, "app.Run()")
	app.Run()
}
