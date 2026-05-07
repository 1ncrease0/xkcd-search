package grpc

import (
	"context"
	"errors"
	"testing"

	updatepb "github.com/1ncrease0/xkcd-search/proto/update"
	"github.com/1ncrease0/xkcd-search/update/adapters/grpc/mocks"
	"github.com/1ncrease0/xkcd-search/update/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServer_Ping(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	resp, err := srv.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestServer_Status_Running(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Status(mock.Anything).Return(core.StatusRunning).Once()

	resp, err := srv.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_RUNNING, resp.Status)
}

func TestServer_Status_Idle(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Status(mock.Anything).Return(core.StatusIdle).Once()

	resp, err := srv.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_IDLE, resp.Status)
}

func TestServer_Status_Unknown(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Status(mock.Anything).Return("unknown").Once()

	resp, err := srv.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_UNSPECIFIED, resp.Status)
}

func TestServer_Update_Success(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Update(mock.Anything).Return(nil).Once()

	resp, err := srv.Update(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestServer_Update_AlreadyExists(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Update(mock.Anything).Return(core.ErrAlreadyExists).Once()

	_, err := srv.Update(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestServer_Update_InternalError(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Update(mock.Anything).Return(errors.New("boom")).Once()

	_, err := srv.Update(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestServer_Stats_Success(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	input := core.ServiceStats{
		DBStats: core.DBStats{
			WordsTotal:    1,
			WordsUnique:   2,
			ComicsFetched: 3,
		},
		ComicsTotal: 123,
	}
	updater.EXPECT().Stats(mock.Anything).Return(input, nil).Once()

	resp, err := srv.Stats(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.EqualExportedValues(t, &updatepb.StatsReply{
		WordsTotal:    int64(input.WordsTotal),
		WordsUnique:   int64(input.WordsUnique),
		ComicsFetched: int64(input.ComicsFetched),
		ComicsTotal:   int64(input.ComicsTotal),
	}, resp)
}

func TestServer_Stats_Error(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Stats(mock.Anything).Return(core.ServiceStats{}, errors.New("stats fail")).Once()

	_, err := srv.Stats(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestServer_Drop_Success(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Drop(mock.Anything).Return(nil).Once()

	resp, err := srv.Drop(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestServer_Drop_AlreadyExists(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Drop(mock.Anything).Return(core.ErrAlreadyExists).Once()

	_, err := srv.Drop(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestServer_Drop_InternalError(t *testing.T) {
	t.Parallel()

	updater := mocks.NewMockUpdater(t)
	srv := NewServer(updater)

	updater.EXPECT().Drop(mock.Anything).Return(errors.New("drop fail")).Once()

	_, err := srv.Drop(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}
