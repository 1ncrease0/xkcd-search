package initiator_test

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/1ncrease0/xkcd-search/search/adapters/initiator"
	"github.com/1ncrease0/xkcd-search/search/core"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type recordingSearcher struct {
	buildIndexCalls atomic.Int32
}

func (r *recordingSearcher) Search(context.Context, string, int) ([]core.Comics, error) {
	return nil, nil
}

func (r *recordingSearcher) ISearch(context.Context, string, int) ([]core.Comics, error) {
	return nil, nil
}

func (r *recordingSearcher) BuildIndex(context.Context) error {
	r.buildIndexCalls.Add(1)
	return nil
}

func TestStart_RunsInitialBuildIndex(t *testing.T) {
	t.Parallel()

	searcher := &recordingSearcher{}
	inf := initiator.NewInitiator(testLogger(), time.Hour, searcher)
	inf.Start(context.Background())

	require.Eventually(t, func() bool {
		return searcher.buildIndexCalls.Load() >= 1
	}, 200*time.Millisecond, 5*time.Millisecond, "initial BuildIndex")
}

func TestStart_RunsPeriodicBuildIndex(t *testing.T) {
	t.Parallel()

	searcher := &recordingSearcher{}
	ttl := 10 * time.Millisecond
	inf := initiator.NewInitiator(testLogger(), ttl, searcher)
	inf.Start(context.Background())

	require.Eventually(t, func() bool {
		return searcher.buildIndexCalls.Load() >= 2
	}, 500*time.Millisecond, 5*time.Millisecond, "tick BuildIndex")
}

func TestStart_StopsWhenContextDone(t *testing.T) {
	t.Parallel()

	searcher := &recordingSearcher{}
	ctx, cancel := context.WithCancel(context.Background())
	inf := initiator.NewInitiator(testLogger(), time.Hour, searcher)
	inf.Start(ctx)

	require.Eventually(t, func() bool {
		return searcher.buildIndexCalls.Load() >= 1
	}, 200*time.Millisecond, 5*time.Millisecond, "first BuildIndex")

	n := searcher.buildIndexCalls.Load()
	cancel()
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, n, searcher.buildIndexCalls.Load(), "stable after cancel")
}
