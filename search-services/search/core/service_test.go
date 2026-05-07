package core_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/1ncrease0/xkcd-search/search/core"
	"github.com/1ncrease0/xkcd-search/search/core/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testPhrase = "linux"

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestService_Search_NormError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	words.EXPECT().Norm(mock.Anything, testPhrase).Return(nil, errors.New("norm failed")).Once()

	_, err := svc.Search(context.Background(), testPhrase, 10)
	require.Error(t, err)
}

func TestService_Search_DBSearchError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	words.EXPECT().Norm(mock.Anything, testPhrase).Return([]string{testPhrase}, nil).Once()
	db.EXPECT().Search(mock.Anything, testPhrase).Return(nil, errors.New("search failed")).Once()

	_, err := svc.Search(context.Background(), testPhrase, 10)
	require.Error(t, err)
}

func TestService_Search_DBGetError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	words.EXPECT().Norm(mock.Anything, testPhrase).Return([]string{testPhrase}, nil).Once()
	db.EXPECT().Search(mock.Anything, testPhrase).Return([]int{1}, nil).Once()
	db.EXPECT().Get(mock.Anything, 1).Return(core.Comics{}, errors.New("get failed")).Once()

	_, err := svc.Search(context.Background(), testPhrase, 10)
	require.Error(t, err)
}

func TestService_Search_SuccessWithLimitAndRanking(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	words.EXPECT().Norm(mock.Anything, "query").Return([]string{"a", "b"}, nil).Once()
	db.EXPECT().Search(mock.Anything, "a").Return([]int{2, 1}, nil).Once()
	db.EXPECT().Search(mock.Anything, "b").Return([]int{2}, nil).Once()
	db.EXPECT().Get(mock.Anything, 2).Return(core.Comics{ID: 2, URL: "u2"}, nil).Once()

	got, err := svc.Search(context.Background(), "query", 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, 2, got[0].ID)
}

func TestService_BuildIndex_DBListError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	expectedErr := errors.New("list failed")
	db.EXPECT().List(mock.Anything).Return(nil, expectedErr).Once()

	err := svc.BuildIndex(context.Background())
	require.ErrorIs(t, err, expectedErr)
}

func TestService_BuildIndex_SuccessDeduplicates(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	db.EXPECT().List(mock.Anything).Return([]core.ComicsWithWords{
		{ID: 1, URL: "u1", Words: []string{"go", "go", "lang"}},
		{ID: 2, URL: "u2", Words: []string{"go"}},
	}, nil).Once()

	err := svc.BuildIndex(context.Background())
	require.NoError(t, err)

	words.EXPECT().Norm(mock.Anything, "go").Return([]string{"go"}, nil).Once()
	got, err := svc.ISearch(context.Background(), "go", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestService_ISearch_NormError(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	words.EXPECT().Norm(mock.Anything, "query").Return(nil, errors.New("norm failed")).Once()

	_, err := svc.ISearch(context.Background(), "query", 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "norm failed")
}

func TestService_ISearch_SuccessWithLimitAndRanking(t *testing.T) {
	t.Parallel()

	log := testLogger()
	db := mocks.NewMockDB(t)
	words := mocks.NewMockWords(t)
	svc := core.NewService(log, db, words)

	db.EXPECT().List(mock.Anything).Return([]core.ComicsWithWords{
		{ID: 1, URL: "u1", Words: []string{"a"}},
		{ID: 2, URL: "u2", Words: []string{"a", "b"}},
	}, nil).Once()
	require.NoError(t, svc.BuildIndex(context.Background()))

	words.EXPECT().Norm(mock.Anything, "query").Return([]string{"a", "b"}, nil).Once()
	got, err := svc.ISearch(context.Background(), "query", 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, 2, got[0].ID)
}
