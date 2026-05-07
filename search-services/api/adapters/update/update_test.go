package update

import (
	"context"
	"errors"
	"testing"

	"github.com/1ncrease0/xkcd-search/api/adapters/update/mocks"
	"github.com/1ncrease0/xkcd-search/api/core"
	updatepb "github.com/1ncrease0/xkcd-search/proto/update"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func newTestClient(t *testing.T) (*Client, *mocks.MockUpdateClient) {
	t.Helper()
	m := mocks.NewMockUpdateClient(t)
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

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(&emptypb.Empty{}, nil).
			Once()

		require.NoError(t, c.Update(context.Background()))
	})

	t.Run("already exists", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.AlreadyExists, "")).
			Once()

		require.ErrorIs(t, c.Update(context.Background()), core.ErrAlreadyExists)
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unavailable, "")).
			Once()

		err := c.Update(context.Background())
		require.Error(t, err)
		require.NotErrorIs(t, err, core.ErrAlreadyExists)
	})
}

func TestStats(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Stats(mock.Anything, mock.Anything).
			Return(&updatepb.StatsReply{
				WordsTotal: 1, WordsUnique: 2, ComicsFetched: 3, ComicsTotal: 4,
			}, nil).
			Once()

		st, err := c.Stats(context.Background())
		require.NoError(t, err)
		require.Equal(t, core.UpdateStats{
			WordsTotal: 1, WordsUnique: 2, ComicsFetched: 3, ComicsTotal: 4,
		}, st)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Stats(mock.Anything, mock.Anything).
			Return(nil, errors.New("db")).
			Once()

		_, err := c.Stats(context.Background())
		require.Error(t, err)
	})
}

func TestStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		pb   updatepb.Status
		want core.UpdateStatus
	}{
		{"running", updatepb.Status_STATUS_RUNNING, core.StatusUpdateRunning},
		{"idle", updatepb.Status_STATUS_IDLE, core.StatusUpdateIdle},
		{"unspecified", updatepb.Status_STATUS_UNSPECIFIED, core.StatusUpdateUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, m := newTestClient(t)
			m.EXPECT().
				Status(mock.Anything, mock.Anything).
				Return(&updatepb.StatusReply{Status: tc.pb}, nil).
				Once()

			got, err := c.Status(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("rpc error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Status(mock.Anything, mock.Anything).
			Return(nil, errors.New("rpc")).
			Once()

		got, err := c.Status(context.Background())
		require.Error(t, err)
		require.Equal(t, core.StatusUpdateUnknown, got)
	})
}

func TestDrop(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Drop(mock.Anything, mock.Anything).
			Return(&emptypb.Empty{}, nil).
			Once()

		require.NoError(t, c.Drop(context.Background()))
	})

	t.Run("already exists", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Drop(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.AlreadyExists, "")).
			Once()

		require.ErrorIs(t, c.Drop(context.Background()), core.ErrAlreadyExists)
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		c, m := newTestClient(t)
		m.EXPECT().
			Drop(mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.FailedPrecondition, "")).
			Once()

		err := c.Drop(context.Background())
		require.Error(t, err)
		require.NotErrorIs(t, err, core.ErrAlreadyExists)
	})
}
