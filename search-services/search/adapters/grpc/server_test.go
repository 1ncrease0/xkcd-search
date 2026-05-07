package grpc

import (
	"context"
	"errors"
	"testing"

	searchpb "github.com/1ncrease0/xkcd-search/proto/search"
	"github.com/1ncrease0/xkcd-search/search/adapters/grpc/mocks"
	"github.com/1ncrease0/xkcd-search/search/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServer_Ping(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	resp, err := srv.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestServer_Search_Success(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().Search(mock.Anything, "linux", 3).Return([]core.Comics{
		{ID: 1, URL: "https://1"},
		{ID: 2, URL: "https://2"},
	}, nil).Once()

	reply, err := srv.Search(context.Background(), &searchpb.SearchRequest{
		Phrase: "linux",
		Limit:  3,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []*searchpb.Comics{
		{Id: 1, Url: "https://1"},
		{Id: 2, Url: "https://2"},
	}, reply.Comics)
}

func TestServer_Search_DefaultLimit(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().Search(mock.Anything, "q", 10).Return([]core.Comics{}, nil).Once()

	_, err := srv.Search(context.Background(), &searchpb.SearchRequest{Phrase: "q", Limit: 0})
	require.NoError(t, err)
}

func TestServer_Search_NotFound(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().Search(mock.Anything, "nope", 10).Return(nil, core.ErrNotFound).Once()

	_, err := srv.Search(context.Background(), &searchpb.SearchRequest{Phrase: "nope", Limit: 0})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestServer_Search_OtherError(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().Search(mock.Anything, "x", 10).Return(nil, errors.New("db down")).Once()

	_, err := srv.Search(context.Background(), &searchpb.SearchRequest{Phrase: "x", Limit: 0})
	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}

func TestServer_ISearch_Success(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().ISearch(mock.Anything, "p", 2).Return([]core.Comics{
		{ID: 7, URL: "u7"},
	}, nil).Once()

	reply, err := srv.ISearch(context.Background(), &searchpb.SearchRequest{Phrase: "p", Limit: 2})
	require.NoError(t, err)
	require.ElementsMatch(t, []*searchpb.Comics{
		{Id: 7, Url: "u7"},
	}, reply.Comics)
}

func TestServer_ISearch_NotFound(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().ISearch(mock.Anything, "nope", 10).Return(nil, core.ErrNotFound).Once()

	_, err := srv.ISearch(context.Background(), &searchpb.SearchRequest{Phrase: "nope", Limit: 0})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestServer_ISearch_OtherError(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	srv := NewServer(s)

	s.EXPECT().ISearch(mock.Anything, "x", 10).Return(nil, errors.New("ix fail")).Once()

	_, err := srv.ISearch(context.Background(), &searchpb.SearchRequest{Phrase: "x", Limit: 0})
	require.Error(t, err)
	require.ErrorContains(t, err, "ix fail")
}
