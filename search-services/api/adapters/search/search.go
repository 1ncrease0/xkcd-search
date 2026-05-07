package search

import (
	"context"
	"log/slog"

	"github.com/1ncrease0/xkcd-search/api/core"
	searchpb "github.com/1ncrease0/xkcd-search/proto/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
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
		client: searchpb.NewSearchClient(conn),
		conn:   conn,
		log:    log,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	reply, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	comics := make([]core.Comics, 0, len(reply.Comics))
	for _, c := range reply.Comics {
		comics = append(comics, core.Comics{ID: int(c.Id), URL: c.Url})
	}
	return comics, nil
}

func (c *Client) ISearch(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	reply, err := c.client.ISearch(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	comics := make([]core.Comics, 0, len(reply.Comics))
	for _, c := range reply.Comics {
		comics = append(comics, core.Comics{ID: int(c.Id), URL: c.Url})
	}
	return comics, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
