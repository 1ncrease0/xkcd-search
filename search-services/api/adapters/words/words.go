package words

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	wordspb "github.com/1ncrease0/xkcd-search/proto/words"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

var connectParams = grpc.ConnectParams{
	Backoff: backoff.Config{
		BaseDelay:  1 * time.Second,
		Multiplier: 1.6,
		MaxDelay:   10 * time.Second,
	},
	MinConnectTimeout: 2 * time.Second,
}

var retryPolicy = `
{
	"methodConfig": [{
		"name": [{"service": "search.Words", "method": ""}],
		"retryPolicy": {
			"maxAttempts": 4,
			"initialBackoff": "1s",
			"maxBackoff": "5s",
			"backoffMultiplier": 2.0,
			"retryableStatusCodes": ["UNAVAILABLE", "INTERNAL", "UNKNOWN"]
		}
	}]
}`

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(connectParams),
		grpc.WithDefaultServiceConfig(retryPolicy),
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

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}
