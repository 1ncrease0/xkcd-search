package words

import (
	"context"
	"log/slog"

	wordspb "github.com/1ncrease0/xkcd-search/proto/words"
	"github.com/1ncrease0/xkcd-search/search/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: wordspb.NewWordsClient(conn),
		conn:   conn,
		log:    log,
	}, nil
}

func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	reply, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase})
	if err != nil {
		if status.Code(err) == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}
		return nil, err
	}
	return reply.GetWords(), nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
