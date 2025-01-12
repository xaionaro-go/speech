package client

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/consts"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/goconv"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/server/proto/go/speechtotext_grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	RemoteAddr    string
	SSTClient     speechtotext_grpc.SpeechToTextClient
	Connection    *grpc.ClientConn
	ContextID     uint64
	ContextHolder speechtotext_grpc.SpeechToText_NewContextClient
	AudioWriter   speechtotext_grpc.SpeechToText_WriteAudioClient
}

var _ speech.ToText = (*Client)(nil)

func New(
	ctx context.Context,
	addr string,
	contextParams *speechtotext_grpc.NewContextRequest,
) (*Client, error) {
	c := &Client{
		RemoteAddr: addr,
	}
	sstClient, conn, err := c.grpcClient()
	if err != nil {
		return nil, err
	}
	c.SSTClient = sstClient
	c.Connection = conn
	ctxClient, err := c.SSTClient.NewContext(ctx, contextParams)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("unable to get a new context: %w", err)
	}
	ctxReply, err := ctxClient.Recv()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("unable to receive the context ID: %w", err)
	}
	c.ContextID = ctxReply.GetContextID()
	c.ContextHolder = ctxClient
	audioWriter, err := c.SSTClient.WriteAudio(ctx)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("unable to initialize audio writer: %w", err)
	}
	c.AudioWriter = audioWriter
	return c, nil
}

func (c *Client) grpcClient() (speechtotext_grpc.SpeechToTextClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		c.RemoteAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(consts.MaxMessageSize), grpc.MaxCallRecvMsgSize(consts.MaxMessageSize)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to initialize a gRPC client: %w", err)
	}

	client := speechtotext_grpc.NewSpeechToTextClient(conn)
	return client, conn, nil
}

func (c *Client) Close() error {
	return c.Connection.Close()
}

func (c *Client) AudioEncoding(context.Context) (audio.Encoding, error) {
	return (*whisper.SpeechToText)(nil).AudioEncodingNoErr(), nil // TODO: request the server to provide this value
}

func (c *Client) AudioChannels(context.Context) (audio.Channel, error) {
	return (*whisper.SpeechToText)(nil).AudioChannelsNoErr(), nil // TODO: request the server to provide this value
}

func (c *Client) WriteAudio(
	ctx context.Context,
	b []byte,
) error {
	return c.AudioWriter.Send(&speechtotext_grpc.WriteAudioRequest{
		ContextID: c.ContextID,
		Audio:     b,
	})
}

func (c *Client) OutputChan(ctx context.Context) (<-chan *speech.Transcript, error) {
	client, err := c.SSTClient.OutputChan(ctx, &speechtotext_grpc.OutputChanRequest{
		ContextID: c.ContextID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to request a channel: %w", err)
	}

	result := make(chan *speech.Transcript, 1)

	observability.Go(ctx, func() {
		for {
			msg, err := client.Recv()
			if err != nil {
				logger.Errorf(ctx, "unable to receive a message from the client: %v", err)
				close(result)
				return
			}

			result <- goconv.TranscriptFromGRPC(msg.GetTranscript())
		}
	})
	return result, nil
}
