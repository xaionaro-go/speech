package main

import (
	"context"
	"io"
	"net"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/subtitleswindow"
)

func main() {
	loggerLevel := logger.LevelDebug
	pflag.Var(&loggerLevel, "log-level", "Log level")
	pflag.Parse()

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	assertNoError(err)

	srv := NewServer()
	observability.Go(ctx, func() {
		srv.Serve(ctx, listener)
	})
	app := app.New()
	r, _ := io.Pipe()
	w, err := subtitleswindow.New(ctx, app, "Fake Subtitles", fyne.TextAlignCenter, r, listener.Addr().String(), 0, nil, "", false, nil, 0)
	if err != nil {
		panic(err)
	}
	w.Show()
	app.Run()
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
