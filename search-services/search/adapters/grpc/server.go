package grpc

import (
	"context"
	"errors"

	searchpb "github.com/1ncrease0/xkcd-search/proto/search"
	"github.com/1ncrease0/xkcd-search/search/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const defaultLimit = 10

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Searcher
}

func (s *Server) Ping(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Search(
	ctx context.Context, req *searchpb.SearchRequest,
) (*searchpb.SearchReply, error) {
	if req.Limit == 0 {
		req.Limit = defaultLimit
	}
	results, err := s.service.Search(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
		return nil, err
	}
	comics := make([]*searchpb.Comics, 0, len(results))
	for _, c := range results {
		comics = append(comics, &searchpb.Comics{
			Id:  int64(c.ID),
			Url: c.URL,
		})
	}
	return &searchpb.SearchReply{Comics: comics}, nil
}

func (s *Server) ISearch(
	ctx context.Context, req *searchpb.SearchRequest,
) (*searchpb.SearchReply, error) {
	if req.Limit == 0 {
		req.Limit = defaultLimit
	}
	results, err := s.service.ISearch(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
		return nil, err
	}
	comics := make([]*searchpb.Comics, 0, len(results))
	for _, c := range results {
		comics = append(comics, &searchpb.Comics{
			Id:  int64(c.ID),
			Url: c.URL,
		})
	}
	return &searchpb.SearchReply{Comics: comics}, nil
}
