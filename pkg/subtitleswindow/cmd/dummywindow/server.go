package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/consts"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
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
}

type objectHash [64 + sha512.Size]byte

func NewServer() *Server {
	srv := &Server{
		GRPCServer: grpc.NewServer(grpc.MaxRecvMsgSize(consts.MaxMessageSize)),
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

	err := respSrv.Send(&speechtotext_grpc.NewContextReply{
		ContextID: 1,
	})
	if err != nil {
		return status.Errorf(codes.Aborted, "unable to send the context ID back to the client: %v", err)
	}

	logger.Debugf(ctx, "initialized context %d", 1)
	<-ctx.Done()
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

		// doing nothing
	}
}

func (srv *Server) OutputChan(
	req *speechtotext_grpc.OutputChanRequest,
	replySrv speechtotext_grpc.SpeechToText_OutputChanServer,
) error {
	ctx := srv.ctx(replySrv.Context())

	t := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-t.C:
			if !ok {
				return status.Errorf(codes.Aborted, "the channel is closed")
			}
			err := replySrv.Send(&speechtotext_grpc.OutputChanReply{
				Transcript: &speechtotext_grpc.Transcript{
					Variants: []*speechtotext_grpc.TranscriptVariant{{
						Text: ev.String(),
					}},
					IsFinal: true,
				},
			})
			if err != nil {
				return status.Errorf(codes.Aborted, "unable to send the transcript to the client: %v", err)
			}
		}
	}
}
