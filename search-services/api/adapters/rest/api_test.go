package rest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/1ncrease0/xkcd-search/api/adapters/rest"
	"github.com/1ncrease0/xkcd-search/api/adapters/rest/mocks"
	"github.com/1ncrease0/xkcd-search/api/core"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func copyMatchingFields(dst, src any) {
	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		panic("dst must be a non-nil pointer")
	}
	dstElem := dstVal.Elem()
	srcType := srcVal.Type()

	for i := range srcVal.NumField() {
		field := srcType.Field(i)
		dstField := dstElem.FieldByName(field.Name)
		if dstField.IsValid() && dstField.CanSet() && dstField.Type() == srcVal.Field(i).Type() {
			dstField.Set(srcVal.Field(i))
		}
	}
}

func searchRequestPath(path, phrase, limit string) string {
	values := url.Values{}
	if phrase != "" {
		values.Set("phrase", phrase)
	}
	if limit != "" {
		values.Set("limit", limit)
	}
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func TestLoginHandler_InvalidBody(t *testing.T) {
	t.Parallel()

	auth := mocks.NewMockAuthenticator(t)
	h := rest.NewLoginHandler(testLogger(), auth)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("not json"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLoginHandler_Unauthorized(t *testing.T) {
	t.Parallel()

	auth := mocks.NewMockAuthenticator(t)
	body, err := json.Marshal(struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{Name: "u", Password: "p"})
	require.NoError(t, err)

	auth.EXPECT().Login("u", "p").Return("", core.ErrUnauthorized).Once()
	h := rest.NewLoginHandler(testLogger(), auth)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLoginHandler_InternalError(t *testing.T) {
	t.Parallel()

	auth := mocks.NewMockAuthenticator(t)
	body, err := json.Marshal(struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{Name: "u", Password: "p"})
	require.NoError(t, err)

	auth.EXPECT().Login("u", "p").Return("", errors.New("backend down")).Once()
	h := rest.NewLoginHandler(testLogger(), auth)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestLoginHandler_OK(t *testing.T) {
	t.Parallel()

	auth := mocks.NewMockAuthenticator(t)
	body, err := json.Marshal(struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{Name: "alice", Password: "s"})
	require.NoError(t, err)

	const wantToken = "jwt-token"
	auth.EXPECT().Login("alice", "s").Return(wantToken, nil).Once()
	h := rest.NewLoginHandler(testLogger(), auth)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	require.Equal(t, wantToken, rr.Body.String())
}

func TestPingHandler_Empty(t *testing.T) {
	t.Parallel()

	h := rest.NewPingHandler(testLogger(), map[string]core.Pinger{})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ping", nil))

	require.Equal(t, http.StatusOK, rr.Code)
	var got rest.PingReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Empty(t, got.Replies)
}

func TestPingHandler_OK(t *testing.T) {
	t.Parallel()

	p := mocks.NewMockPinger(t)
	p.EXPECT().Ping(mock.Anything).Return(nil).Once()

	h := rest.NewPingHandler(
		testLogger(),
		map[string]core.Pinger{"words": p},
	)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ping", nil))

	require.Equal(t, http.StatusOK, rr.Code)
	var got rest.PingReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, map[string]string{"words": "ok"}, got.Replies)
}

func TestPingHandler_Unavailable(t *testing.T) {
	t.Parallel()

	p := mocks.NewMockPinger(t)
	p.EXPECT().Ping(mock.Anything).Return(errors.New("down")).Once()

	h := rest.NewPingHandler(
		testLogger(),
		map[string]core.Pinger{"search": p},
	)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ping", nil))

	require.Equal(t, http.StatusOK, rr.Code)
	var got rest.PingReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, map[string]string{"search": "unavailable"}, got.Replies)
}

func TestPingHandler_OneUpOneDown(t *testing.T) {
	t.Parallel()

	good := mocks.NewMockPinger(t)
	bad := mocks.NewMockPinger(t)
	good.EXPECT().Ping(mock.Anything).Return(nil).Once()
	bad.EXPECT().Ping(mock.Anything).Return(errors.New("down")).Once()

	h := rest.NewPingHandler(
		testLogger(),
		map[string]core.Pinger{
			"good": good,
			"bad":  bad,
		},
	)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ping", nil))

	require.Equal(t, http.StatusOK, rr.Code)
	var got rest.PingReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, "ok", got.Replies["good"])
	require.Equal(t, "unavailable", got.Replies["bad"])
}

func TestUpdateHandler_OK(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Update(mock.Anything).Return(nil).Once()
	h := rest.NewUpdateHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/update", nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	require.Empty(t, rr.Body.String())
}

func TestUpdateHandler_AlreadyExists(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Update(mock.Anything).Return(core.ErrAlreadyExists).Once()
	h := rest.NewUpdateHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/update", nil))

	require.Equal(t, http.StatusAccepted, rr.Code)
	require.Equal(t, core.ErrAlreadyExists.Error()+"\n", rr.Body.String())
}

func TestUpdateHandler_InternalError(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Update(mock.Anything).Return(errors.New("upstream failed")).Once()
	h := rest.NewUpdateHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/update", nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "upstream failed\n", rr.Body.String())
}

func TestUpdateStatsHandler_OK(t *testing.T) {
	t.Parallel()

	input := core.UpdateStats{
		WordsTotal:    1,
		WordsUnique:   2,
		ComicsFetched: 3,
		ComicsTotal:   4,
	}

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Stats(mock.Anything).Return(input, nil).Once()
	h := rest.NewUpdateStatsHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/update/stats", nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	var got rest.StatsReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	var want rest.StatsReply
	copyMatchingFields(&want, input)
	require.Equal(t, want, got)
}

func TestUpdateStatsHandler_Error(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Stats(mock.Anything).Return(core.UpdateStats{}, errors.New("db down")).Once()
	h := rest.NewUpdateStatsHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/stats", nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestUpdateStatusHandler_OK(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Status(mock.Anything).Return(core.StatusUpdateRunning, nil).Once()
	h := rest.NewUpdateStatusHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	var got rest.StatusReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, rest.StatusReply{Status: "running"}, got)
}

func TestUpdateStatusHandler_Error(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Status(mock.Anything).Return(core.StatusUpdateUnknown, errors.New("down")).Once()
	h := rest.NewUpdateStatusHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestDropHandler_OK(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Drop(mock.Anything).Return(nil).Once()
	h := rest.NewDropHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/drop", nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	require.Empty(t, rr.Body.String())
}

func TestDropHandler_Conflict(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Drop(mock.Anything).Return(core.ErrAlreadyExists).Once()
	h := rest.NewDropHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/drop", nil))

	require.Equal(t, http.StatusConflict, rr.Code)
	require.Equal(t, core.ErrAlreadyExists.Error()+"\n", rr.Body.String())
}

func TestDropHandler_InternalError(t *testing.T) {
	t.Parallel()

	u := mocks.NewMockUpdater(t)
	u.EXPECT().Drop(mock.Anything).Return(errors.New("drop failed")).Once()
	h := rest.NewDropHandler(testLogger(), u)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/drop", nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestSearchHandler_MissingPhrase(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "", ""), nil))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "missing phrase\n", rr.Body.String())
}

func TestSearchHandler_BadLimit_NaN(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "x", "nope"), nil))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "bad limit\n", rr.Body.String())
}

func TestSearchHandler_BadLimit_Zero(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "x", "0"), nil))

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSearchHandler_NotFound(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	s.EXPECT().Search(mock.Anything, "q", 0).Return(nil, core.ErrNotFound).Once()
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "q", ""), nil))

	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Equal(t, "comics not founded\n", rr.Body.String())
}

func TestSearchHandler_InternalError(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	s.EXPECT().Search(mock.Anything, "q", 0).Return(nil, errors.New("rpc")).Once()
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "q", ""), nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "rpc\n", rr.Body.String())
}

func TestSearchHandler_OK(t *testing.T) {
	t.Parallel()

	input := []core.Comics{{ID: 1, URL: "u"}}

	s := mocks.NewMockSearcher(t)
	s.EXPECT().Search(mock.Anything, "q", 0).Return(input, nil).Once()
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "q", ""), nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	var got rest.SearchReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	wantComics := make([]rest.ComicsReply, 0, len(input))
	for _, c := range input {
		var item rest.ComicsReply
		copyMatchingFields(&item, c)
		wantComics = append(wantComics, item)
	}
	want := rest.SearchReply{
		Comics: wantComics,
		Total:  len(wantComics),
	}
	require.Equal(t, want, got)
}

func TestSearchHandler_OK_WithLimit(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	s.EXPECT().Search(mock.Anything, "q", 3).Return([]core.Comics{}, nil).Once()
	h := rest.NewSearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/search", "q", "3"), nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

func TestISearchHandler_MissingPhrase(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	h := rest.NewISearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/isearch", "", ""), nil))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "missing phrase\n", rr.Body.String())
}

func TestISearchHandler_BadLimit_NaN(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	h := rest.NewISearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/isearch", "x", "nope"), nil))

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestISearchHandler_NotFound(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	s.EXPECT().ISearch(mock.Anything, "q", 0).Return(nil, core.ErrNotFound).Once()
	h := rest.NewISearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/isearch", "q", ""), nil))

	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Equal(t, "comics not founded\n", rr.Body.String())
}

func TestISearchHandler_InternalError(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	s.EXPECT().ISearch(mock.Anything, "q", 0).Return(nil, errors.New("rpc")).Once()
	h := rest.NewISearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/isearch", "q", ""), nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "rpc\n", rr.Body.String())
}

func TestISearchHandler_OK(t *testing.T) {
	t.Parallel()

	input := []core.Comics{{ID: 2, URL: "v"}}

	s := mocks.NewMockSearcher(t)
	s.EXPECT().ISearch(mock.Anything, "q", 0).Return(input, nil).Once()
	h := rest.NewISearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/isearch", "q", ""), nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	var got rest.SearchReply
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))

	wantComics := make([]rest.ComicsReply, 0, len(input))
	for _, c := range input {
		var item rest.ComicsReply
		copyMatchingFields(&item, c)
		wantComics = append(wantComics, item)
	}
	want := rest.SearchReply{
		Comics: wantComics,
		Total:  len(wantComics),
	}
	require.Equal(t, want, got)
}

func TestISearchHandler_OK_WithLimit(t *testing.T) {
	t.Parallel()

	s := mocks.NewMockSearcher(t)
	s.EXPECT().ISearch(mock.Anything, "a", 5).Return([]core.Comics{}, nil).Once()
	h := rest.NewISearchHandler(testLogger(), s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, searchRequestPath("/isearch", "a", "5"), nil))

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
}
