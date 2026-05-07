package words

import (
	"context"
	"errors"
	"testing"

	"github.com/1ncrease0/xkcd-search/api/adapters/words/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func newTestClient(t *testing.T) (*Client, *mocks.MockWordsClient) {
	t.Helper()
	m := mocks.NewMockWordsClient(t)
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
