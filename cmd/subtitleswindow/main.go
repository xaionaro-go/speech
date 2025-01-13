package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/audio/pkg/audio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/pulseaudio"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/player/pkg/player/builtin"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
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
	playbackFlag := pflag.Bool("audio-loopback", false, "[debug] instead of running a subtitles window, playback the audio")
	remoteFlag := pflag.String("remote-addr", "", "use a remote speech-to-text engine, instead of running it locally")
	textAlignmentFlag := pflag.String("text-align", "center", "allowed values: left, center, right")
	gpuFlag := pflag.Int("gpu", -1, "")
	pflag.Parse()
	if pflag.NArg() < 1 || pflag.NArg() > 2 {
		syntaxExit("expected one or two arguments: whisper-model-path [input]")
	}

	whisperModelPath := pflag.Arg(0)

	var mediaURL string

	if pflag.NArg() == 2 {
		mediaURL = pflag.Arg(1)
	}

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	if *netPprofAddr != "" {
		observability.Go(ctx, func() { l.Error(http.ListenAndServe(*netPprofAddr, nil)) })
	}

	var textAlignment fyne.TextAlign
	switch *textAlignmentFlag {
	case "left":
		textAlignment = fyne.TextAlignLeading
	case "center":
		textAlignment = fyne.TextAlignCenter
	case "right":
		textAlignment = fyne.TextAlignTrailing
	}

	var whisperModel []byte
	if *remoteFlag == "" || whisperModelPath != "" {
		var err error
		whisperModel, err = os.ReadFile(whisperModelPath)
		if err != nil {
			panic(err)
		}
	}

	audioEnc := (*whisper.SpeechToText)(nil).AudioEncodingNoErr().(audio.EncodingPCM)
	audioChannels := (*whisper.SpeechToText)(nil).AudioChannelsNoErr()

	var audioInput io.Reader
	if mediaURL == "" {
		r, w := io.Pipe()
		recorder := audio.NewRecorderAuto(ctx)
		logger.Infof(ctx, "using %T as the audio input", recorder.RecorderPCM)
		stream, err := recorder.RecordPCM(audioEnc.SampleRate, audioChannels, audioEnc.PCMFormat, w)
		if err != nil {
			panic(err)
		}
		audioInput = r
		defer func() {
			stream.Close()
		}()
	} else {
		rcv := subtitleswindow.NewDummyPCMPlayer(ctx)
		mediaPlayer := builtin.New(ctx, nil, rcv)
		logger.Debugf(ctx, "builtin.New(ctx, nil, rcv)")

		err := mediaPlayer.OpenURL(ctx, mediaURL)
		if err != nil {
			panic(err)
		}

		audioInput = rcv
	}

	if *playbackFlag {
		player := audio.NewPlayerAuto(ctx)
		logger.Infof(ctx, "using %T as the audio output", player.PlayerPCM)
		stream, err := player.PlayPCM(audioEnc.SampleRate, audioChannels, audioEnc.PCMFormat, time.Millisecond*100, audioInput)
		if err != nil {
			panic(err)
		}
		stream.Drain()
		<-ctx.Done()
		os.Exit(0)
	}

	app := app.New()
	w, err := subtitleswindow.New(ctx, app, "Subtitles", textAlignment, audioInput, *remoteFlag, *gpuFlag, whisperModel, speech.Language(*langFlag), *shouldTranslateFlag)
	if err != nil {
		panic(err)
	}
	w.Show()
	logger.Debugf(ctx, "app.Run()")
	app.Run()
}
