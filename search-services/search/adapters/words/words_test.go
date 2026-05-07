package words

import (
	"context"
	"io"
	"log/slog"
	"testing"

	wordspb "github.com/1ncrease0/xkcd-search/proto/words"
	"github.com/1ncrease0/xkcd-search/search/core"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeWordsClient struct {
	normReply *wordspb.WordsReply
	normErr   error
}

func (f *fakeWordsClient) Ping(context.Context, *emptypb.Empty, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fakeWordsClient) Norm(_ context.Context, _ *wordspb.WordsRequest, _ ...grpc.CallOption) (*wordspb.WordsReply, error) {
	return f.normReply, f.normErr
}

func TestClient_Norm_Success(t *testing.T) {
	t.Parallel()
	c := &Client{
		log: testLogger(),
		client: &fakeWordsClient{
			normReply: &wordspb.WordsReply{Words: []string{"a", "b"}},
		},
	}
	out, err := c.Norm(context.Background(), "hello")
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, out)
}

func TestClient_Norm_ResourceExhausted_ErrBadArguments(t *testing.T) {
	t.Parallel()
	c := &Client{
		log: testLogger(),
		client: &fakeWordsClient{
			normErr: status.Error(codes.ResourceExhausted, "too big"),
		},
	}
	_, err := c.Norm(context.Background(), "x")
	require.ErrorIs(t, err, core.ErrBadArguments)
}

func TestClient_Norm_OtherError(t *testing.T) {
	t.Parallel()
	c := &Client{
		log: testLogger(),
		client: &fakeWordsClient{
			normErr: status.Error(codes.Internal, "boom"),
		},
	}
	_, err := c.Norm(context.Background(), "x")
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}
