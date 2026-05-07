package grpc

import (
	"context"
	"strconv"

	wordspb "github.com/1ncrease0/xkcd-search/proto/words"
	"github.com/1ncrease0/xkcd-search/words/words"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const maxPhraseLen = 20000

type Server struct {
	wordspb.UnimplementedWordsServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Ping(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Norm(ctx context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if len(in.GetPhrase()) > maxPhraseLen {
		return nil, status.Error(
			codes.ResourceExhausted,
			"phrase is large than "+strconv.Itoa(maxPhraseLen),
		)
	}
	return &wordspb.WordsReply{
		Words: words.Norm(in.GetPhrase()),
	}, nil
}
