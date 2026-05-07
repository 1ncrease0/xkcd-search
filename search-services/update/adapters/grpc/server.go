package grpc

import (
	"context"
	"errors"

	updatepb "github.com/1ncrease0/xkcd-search/proto/update"
	"github.com/1ncrease0/xkcd-search/update/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Status(ctx context.Context, in *emptypb.Empty) (*updatepb.StatusReply, error) {
	var pbStatus updatepb.Status
	switch s.service.Status(ctx) {
	case core.StatusRunning:
		pbStatus = updatepb.Status_STATUS_RUNNING
	case core.StatusIdle:
		pbStatus = updatepb.Status_STATUS_IDLE
	default:
		pbStatus = updatepb.Status_STATUS_UNSPECIFIED
	}

	return &updatepb.StatusReply{Status: pbStatus}, nil
}

func (s *Server) Update(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) Stats(ctx context.Context, in *emptypb.Empty) (*updatepb.StatsReply, error) {
	stats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &updatepb.StatsReply{
		WordsTotal:    int64(stats.WordsTotal),
		WordsUnique:   int64(stats.WordsUnique),
		ComicsTotal:   int64(stats.ComicsTotal),
		ComicsFetched: int64(stats.ComicsFetched),
	}, nil
}

func (s *Server) Drop(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}
