package core_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/1ncrease0/xkcd-search/update/core"
	"github.com/1ncrease0/xkcd-search/update/core/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type serviceTestDeps struct {
	svc      core.Updater
	db       *mocks.MockDB
	xkcd     *mocks.MockXKCD
	words    *mocks.MockWords
	notifier *mocks.MockUpdateNotifier
}

func newServiceTestDeps(t *testing.T, concurrency int) serviceTestDeps {
	t.Helper()

	deps := serviceTestDeps{
		db:       mocks.NewMockDB(t),
		xkcd:     mocks.NewMockXKCD(t),
		words:    mocks.NewMockWords(t),
		notifier: mocks.NewMockUpdateNotifier(t),
	}

	svc, err := core.NewService(testLogger(), deps.db, deps.xkcd, deps.words, deps.notifier, concurrency)
	require.NoError(t, err)
	deps.svc = svc

	return deps
}

func TestNewService_InvalidConcurrency(t *testing.T) {
	t.Parallel()
	log := testLogger()
	_, err := core.NewService(log, nil, nil, nil, nil, 0)
	require.Error(t, err)
}

func TestNewService_ValidConcurrency(t *testing.T) {
	t.Parallel()
	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	s, err := core.NewService(log, db, xkcd, words, notifier, 10)

	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestService_Update_NoMissingIDs(t *testing.T) {
	t.Parallel()
	deps := newServiceTestDeps(t, 1)

	deps.xkcd.EXPECT().LastID(mock.Anything).Return(3, nil).Once()
	deps.db.EXPECT().IDs(mock.Anything).Return([]int{1, 2, 3}, nil).Once()

	err := deps.svc.Update(context.Background())
	require.NoError(t, err)

	deps.notifier.AssertNotCalled(t, "Notify", mock.Anything)
}

func TestService_Update_AlreadyRunning(t *testing.T) {
	deps := newServiceTestDeps(t, 1)

	enteredLastID := make(chan struct{})
	releaseLastID := make(chan struct{})

	deps.xkcd.EXPECT().
		LastID(mock.Anything).
		RunAndReturn(func(context.Context) (int, error) {
			close(enteredLastID)
			<-releaseLastID
			return 1, nil
		}).
		Once()

	deps.db.EXPECT().IDs(mock.Anything).Return([]int{1}, nil).Once()

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- deps.svc.Update(context.Background())
	}()

	<-enteredLastID

	err := deps.svc.Update(context.Background())
	require.ErrorIs(t, err, core.ErrAlreadyExists)

	close(releaseLastID)
	require.NoError(t, <-firstDone)

	deps.notifier.AssertNotCalled(t, "Notify", mock.Anything)
}

func TestService_Update_Success_OneMissingID(t *testing.T) {
	t.Parallel()
	deps := newServiceTestDeps(t, 1)

	deps.xkcd.EXPECT().LastID(mock.Anything).Return(2, nil).Once()
	deps.db.EXPECT().IDs(mock.Anything).Return([]int{1}, nil).Once()

	deps.xkcd.EXPECT().Get(mock.Anything, 2).Return(core.XKCDInfo{
		ID:          2,
		URL:         "https://imgs.xkcd.com/comics/2.jpg",
		Title:       "Go",
		Description: "Tests",
	}, nil).Once()

	deps.words.EXPECT().Norm(mock.Anything, "Go Tests").
		Return([]string{"go", "test"}, nil).
		Once()

	deps.db.EXPECT().Add(mock.Anything, core.Comics{
		ID:    2,
		URL:   "https://imgs.xkcd.com/comics/2.jpg",
		Words: []string{"go", "test"},
	}).Return(nil).Once()

	deps.notifier.EXPECT().Notify(mock.Anything).Return(nil).Once()

	err := deps.svc.Update(context.Background())
	require.NoError(t, err)
}

func TestService_Stats_Success(t *testing.T) {
	t.Parallel()
	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	db.EXPECT().Stats(mock.Anything).Return(core.DBStats{
		WordsTotal:    10,
		WordsUnique:   7,
		ComicsFetched: 2,
	}, nil).Once()
	xkcd.EXPECT().LastID(mock.Anything).Return(100, nil).Once()

	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 10, stats.WordsTotal)
	require.Equal(t, 7, stats.WordsUnique)
	require.Equal(t, 2, stats.ComicsFetched)
	require.Equal(t, 100, stats.ComicsTotal)
}

func TestService_Stats_DBError(t *testing.T) {
	t.Parallel()
	deps := newServiceTestDeps(t, 1)

	deps.db.EXPECT().Stats(mock.Anything).Return(core.DBStats{}, errors.New("err")).Once()

	_, err := deps.svc.Stats(context.Background())
	require.Error(t, err)
}

func TestService_Drop_Success(t *testing.T) {
	t.Parallel()
	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	db.EXPECT().Drop(mock.Anything).Return(nil).Once()
	notifier.EXPECT().Notify(mock.Anything).Return(nil).Once()

	err = svc.Drop(context.Background())
	require.NoError(t, err)
}

func TestService_Drop_DBError(t *testing.T) {
	t.Parallel()
	deps := newServiceTestDeps(t, 1)

	deps.db.EXPECT().Drop(mock.Anything).Return(errors.New("err")).Once()

	err := deps.svc.Drop(context.Background())
	require.Error(t, err)

	deps.notifier.AssertNotCalled(t, "Notify", mock.Anything)
}

func TestService_Update_LastIDError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	xkcd.EXPECT().LastID(mock.Anything).Return(0, errors.New("xkcd unavailable")).Once()

	err = svc.Update(context.Background())
	require.Error(t, err)

	notifier.AssertNotCalled(t, "Notify", mock.Anything)
}

func TestService_Update_IDsError(t *testing.T) {
	t.Parallel()

	deps := newServiceTestDeps(t, 1)

	deps.xkcd.EXPECT().LastID(mock.Anything).Return(3, nil).Once()
	deps.db.EXPECT().IDs(mock.Anything).Return(nil, errors.New("err")).Once()

	err := deps.svc.Update(context.Background())
	require.Error(t, err)

	deps.notifier.AssertNotCalled(t, "Notify", mock.Anything)
}

func TestService_Stats_LastIDError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	db.EXPECT().Stats(mock.Anything).Return(core.DBStats{
		WordsTotal:    10,
		WordsUnique:   7,
		ComicsFetched: 2,
	}, nil).Once()
	xkcd.EXPECT().LastID(mock.Anything).Return(0, errors.New("xkcd down")).Once()

	_, err = svc.Stats(context.Background())
	require.Error(t, err)
}

func TestService_Drop_NotifyError_DoesNotFail(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	db.EXPECT().Drop(mock.Anything).Return(nil).Once()
	notifier.EXPECT().Notify(mock.Anything).Return(errors.New("nats down")).Once()

	err = svc.Drop(context.Background())
	require.NoError(t, err)
}

func TestService_Update_ContextCanceled(t *testing.T) {
	t.Parallel()

	deps := newServiceTestDeps(t, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deps.xkcd.EXPECT().LastID(mock.Anything).Return(2, nil).Once()
	deps.db.EXPECT().IDs(mock.Anything).Return([]int{1}, nil).Once()

	err := deps.svc.Update(ctx)
	require.ErrorIs(t, err, context.Canceled)

	deps.notifier.AssertNotCalled(t, "Notify", mock.Anything)
}

func TestService_Update_GetNotFound_SkipsAndContinues(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	xkcd.EXPECT().LastID(mock.Anything).Return(3, nil).Once()
	db.EXPECT().IDs(mock.Anything).Return([]int{1}, nil).Once()

	xkcd.EXPECT().Get(mock.Anything, 2).Return(core.XKCDInfo{}, core.ErrNotFound).Once()

	xkcd.EXPECT().Get(mock.Anything, 3).Return(core.XKCDInfo{
		ID:          3,
		URL:         "https://example.com/3.png",
		Title:       "Title",
		Description: "Description",
	}, nil).Once()
	words.EXPECT().Norm(mock.Anything, "Title Description").Return([]string{"title", "description"}, nil).Once()
	db.EXPECT().Add(mock.Anything, core.Comics{
		ID:    3,
		URL:   "https://example.com/3.png",
		Words: []string{"title", "description"},
	}).Return(nil).Once()

	notifier.EXPECT().Notify(mock.Anything).Return(nil).Once()

	err = svc.Update(context.Background())
	require.NoError(t, err)
}

func TestService_Stats_ReturnsCorrectData(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	xkcd := mocks.NewMockXKCD(t)
	words := mocks.NewMockWords(t)
	notifier := mocks.NewMockUpdateNotifier(t)

	ctx := context.Background()

	expectedDBStats := core.DBStats{
		WordsTotal:    10,
		WordsUnique:   8,
		ComicsFetched: 6,
	}

	db.EXPECT().Stats(ctx).Return(expectedDBStats, nil).Once()
	xkcd.EXPECT().LastID(ctx).Return(100, nil).Once()

	svc, err := core.NewService(log, db, xkcd, words, notifier, 1)
	require.NoError(t, err)

	stats, err := svc.Stats(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedDBStats, stats.DBStats)
	require.Equal(t, 100, stats.ComicsTotal)
}
