package search

import (
	"context"
	"errors"
	"testing"

	"github.com/1ncrease0/xkcd-search/api/adapters/search/mocks"
	"github.com/1ncrease0/xkcd-search/api/core"
	searchpb "github.com/1ncrease0/xkcd-search/proto/search"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func newTestClient(t *testing.T) (*Client, *mocks.MockSearchClient) {
	t.Helper()
	m := mocks.NewMockSearchClient(t)
	return &Client{client: m}, m
}

func TestPing(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Ping(mock.Anything, mock.Anything).
			Return(&emptypb.Empty{}, nil).
			Once()

		require.NoError(t, c.Ping(context.Background()))
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Ping(mock.Anything, mock.Anything).
			Return(nil, errors.New("rpc down")).
			Once()

		require.Error(t, c.Ping(context.Background()))
	})
}

func TestSearch(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Search(mock.Anything, &searchpb.SearchRequest{Phrase: "p", Limit: 2}).
			Return(&searchpb.SearchReply{
				Comics: []*searchpb.Comics{{Id: 1, Url: "u"}},
			}, nil).
			Once()

		got, err := c.Search(context.Background(), "p", 2)
		require.NoError(t, err)
		require.Equal(t, []core.Comics{{ID: 1, URL: "u"}}, got)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Search(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "")).
			Once()

		_, err := c.Search(context.Background(), "p", 0)
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Search(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "x")).
			Once()

		_, err := c.Search(context.Background(), "p", 0)
		require.Error(t, err)
		require.NotErrorIs(t, err, core.ErrNotFound)
	})
}

func TestISearch(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			ISearch(mock.Anything, &searchpb.SearchRequest{Phrase: "p", Limit: 1}).
			Return(&searchpb.SearchReply{
				Comics: []*searchpb.Comics{{Id: 2, Url: "v"}},
			}, nil).
			Once()

		got, err := c.ISearch(context.Background(), "p", 1)
		require.NoError(t, err)
		require.Equal(t, []core.Comics{{ID: 2, URL: "v"}}, got)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			ISearch(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "")).
			Once()

		_, err := c.ISearch(context.Background(), "p", 0)
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			ISearch(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unavailable, "")).
			Once()

		_, err := c.ISearch(context.Background(), "p", 0)
		require.Error(t, err)
		require.NotErrorIs(t, err, core.ErrNotFound)
	})
}
