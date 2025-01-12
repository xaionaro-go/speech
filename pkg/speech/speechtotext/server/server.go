package server

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/consts"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/goconv"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
	"github.com/xaionaro-go/xslices"
	"github.com/xaionaro-go/xsync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RecoderID uint64
type EncoderID uint64
type InputID uint64
type OutputID uint64

type Server struct {
	speechtotext_grpc.UnimplementedSpeechToTextServer

	GRPCServer *grpc.Server
	IsStarted  bool

	BeltLocker xsync.Mutex
	Belt       *belt.Belt

	NextContextID atomic.Uint64
	ContextMap    sync.Map

	ContextsLimit uint
	Options       Options
}

func NewServer(
	contextsLimit uint,
	opts ...Option,
) *Server {
	srv := &Server{
		GRPCServer:    grpc.NewServer(grpc.MaxRecvMsgSize(consts.MaxMessageSize)),
		ContextsLimit: contextsLimit,
		Options:       opts,
	}
	speechtotext_grpc.RegisterSpeechToTextServer(srv.GRPCServer, srv)
	return srv
}

func (srv *Server) Serve(
	ctx context.Context,
	listener net.Listener,
) error {
	if srv.IsStarted {
		panic("this GRPC server was already started at least once")
	}
	srv.IsStarted = true
	srv.Belt = belt.CtxBelt(ctx)
	return srv.GRPCServer.Serve(listener)
}

func (srv *Server) belt() *belt.Belt {
	ctx := context.TODO()
	return xsync.DoR1(xsync.WithNoLogging(ctx, true), &srv.BeltLocker, func() *belt.Belt {
		return srv.Belt
	})
}

func (srv *Server) ctx(ctx context.Context) context.Context {
	return belt.CtxWithBelt(ctx, srv.belt())
}

func (srv *Server) Ping(
	ctx context.Context,
	req *speechtotext_grpc.PingRequest,
) (*speechtotext_grpc.PingReply, error) {
	ctx = srv.ctx(ctx)
	var payload strings.Builder
	extraSize := req.GetRequestExtraPayloadSize()
	totalSize := len(
		req.GetPayloadToReturn(),
	) + int(
		extraSize,
	)
	if totalSize > 65535 {
		return nil, fmt.Errorf(
			"requested a too big payload",
		)
	}
	payload.WriteString(req.GetPayloadToReturn())
	payload.WriteString(strings.Repeat("0", int(extraSize)))
	return &speechtotext_grpc.PingReply{
		Payload: payload.String(),
	}, nil
}

func (srv *Server) NewContext(
	req *speechtotext_grpc.NewContextRequest,
	respSrv speechtotext_grpc.SpeechToText_NewContextServer,
) error {
	ctx := srv.ctx(respSrv.Context())

	count := 0
	srv.ContextMap.Range(func(key, value any) bool {
		count++
		return true
	})
	if count >= int(srv.ContextsLimit) {
		return status.Errorf(codes.ResourceExhausted, "too many contexts already created, please close previous contexts first")
	}

	modelBytes := xslices.Clone(req.GetModelBytes())

	cfg := srv.Options.config()

	var stt speech.ToText
	backend := req.GetBackend()
	switch backend := backend.(type) {
	case *speechtotext_grpc.NewContextRequest_Whisper:
		var err error
		stt, err = whisper.New(
			ctx,
			modelBytes,
			speech.Language(req.GetLanguage()),
			goconv.SamplingStrategyFromGRPC(backend.Whisper.GetSamplingStrategy()),
			req.GetShouldTranslate(),
			goconv.AlignmentAheadsPresetFromGRPC(backend.Whisper.GetAlignmentAheadsPreset()),
			cfg.WhisperOptions...,
		)
		if err != nil {
			return status.Errorf(codes.Unknown, "unable to initialize a whisper instance: %v", err)
		}
	default:
		return status.Errorf(codes.InvalidArgument, "backend type %T is not supported, yet", backend)
	}

	contextID := srv.NextContextID.Add(1)
	srv.ContextMap.Store(contextID, stt)
	defer func() {
		logger.Debugf(ctx, "closing context %d", contextID)
		srv.ContextMap.Delete(contextID)
		stt.Close()
	}()

	err := respSrv.Send(&speechtotext_grpc.NewContextReply{
		ContextID: contextID,
	})
	if err != nil {
		return status.Errorf(codes.Aborted, "unable to send the context ID back to the client: %v", err)
	}

	logger.Debugf(ctx, "initialized context %d", contextID)
	<-ctx.Done()
	runtime.KeepAlive(modelBytes)
	return ctx.Err()
}

func (srv *Server) WriteAudio(
	reqSrv speechtotext_grpc.SpeechToText_WriteAudioServer,
) error {
	ctx := srv.ctx(reqSrv.Context())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := reqSrv.Recv()
		if err != nil {
			return status.Errorf(codes.Aborted, "unable to receive the audio frame from the client: %v", err)
		}

		contextID := req.GetContextID()
		sttI, ok := srv.ContextMap.Load(contextID)
		if !ok {
			return status.Errorf(codes.NotFound, "there is no open context with ID %d", contextID)
		}
		stt := sttI.(speech.ToText)

		frame := req.GetAudio()
		err = stt.WriteAudio(ctx, frame)
		if err != nil {
			return status.Errorf(codes.Unknown, "unable to write audio of length %d to Whisper: %v", len(frame), err)
		}
	}
}

func (srv *Server) OutputChan(
	req *speechtotext_grpc.OutputChanRequest,
	replySrv speechtotext_grpc.SpeechToText_OutputChanServer,
) error {
	ctx := srv.ctx(replySrv.Context())

	contextID := req.GetContextID()
	sttI, ok := srv.ContextMap.Load(contextID)
	if !ok {
		return status.Errorf(codes.NotFound, "there is no open context with ID %d", contextID)
	}
	stt := sttI.(speech.ToText)

	ch, err := stt.OutputChan(ctx)
	if err != nil {
		return status.Errorf(codes.Unknown, "unable to get the event channel: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ch:
			err := replySrv.Send(&speechtotext_grpc.OutputChanReply{
				Transcript: goconv.TranscriptToGRPC(t),
			})
			if err != nil {
				return status.Errorf(codes.Aborted, "unable to send the transcript to the client: %v", err)
			}
		}
	}
}
