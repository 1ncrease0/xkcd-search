package grpc

import (
	"context"
	"strings"
	"testing"

	wordspb "github.com/1ncrease0/xkcd-search/proto/words"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServer_Ping(t *testing.T) {
	t.Parallel()
	srv := NewServer()
	resp, err := srv.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestServer_Norm_OKPhrase(t *testing.T) {
	t.Parallel()
	srv := NewServer()
	req := &wordspb.WordsRequest{Phrase: "I follow followers"}
	resp, err := srv.Norm(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.ElementsMatch(t, []string{"follow"}, resp.Words)
}
func TestServer_Norm_NilRequest(t *testing.T) {
	t.Parallel()
	srv := NewServer()
	resp, err := srv.Norm(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.ElementsMatch(t, []string{}, resp.Words)
}
func TestServer_Norm_TooLargePhrase(t *testing.T) {
	t.Parallel()
	srv := NewServer()
	req := &wordspb.WordsRequest{
		Phrase: strings.Repeat("a", maxPhraseLen+1),
	}
	resp, err := srv.Norm(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
}
