package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/1ncrease0/xkcd-search/update/core"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewClient_EmptyURL(t *testing.T) {
	t.Parallel()

	const emptyURL = ""
	c, err := NewClient(emptyURL, time.Second, testLogger())
	require.Error(t, err)
	require.Nil(t, c)
}

func TestClient_Get_Success(t *testing.T) {
	t.Parallel()

	const comicID = 5
	expectedPath := fmt.Sprintf("/%d/info.0.json", comicID)
	expected := core.XKCDInfo{
		ID:          comicID,
		URL:         "https://img/5.png",
		Title:       "Safe",
		Description: "TRANSCRIPT",
	}
	payload, err := json.Marshal(map[string]any{
		"num":        comicID,
		"safe_title": expected.Title,
		"title":      "Plain",
		"img":        expected.URL,
		"alt":        "ALT",
		"transcript": expected.Description,
	})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, expectedPath, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	got, err := c.Get(context.Background(), comicID)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestClient_Get_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	_, err = c.Get(context.Background(), 404)
	require.Error(t, err)
	require.ErrorIs(t, err, core.ErrNotFound)
}

func TestClient_Get_Fallbacks(t *testing.T) {
	t.Parallel()

	const comicID = 7
	const expectedTitle = "Plain"
	const expectedDescription = "ALT"
	payload, err := json.Marshal(map[string]any{
		"num":        comicID,
		"safe_title": "",
		"title":      expectedTitle,
		"img":        "https://img/7.png",
		"alt":        expectedDescription,
		"transcript": "",
	})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	got, err := c.Get(context.Background(), comicID)
	require.NoError(t, err)
	require.Equal(t, expectedTitle, got.Title)
	require.Equal(t, expectedDescription, got.Description)
}

func TestClient_Get_BadStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	_, err = c.Get(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status")
}

func TestClient_Get_BadJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	_, err = c.Get(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode response")
}

func TestClient_LastID_Success(t *testing.T) {
	t.Parallel()

	const expectedLastID = 321
	payload, err := json.Marshal(map[string]any{
		"num": expectedLastID,
	})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/info.0.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	id, err := c.LastID(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedLastID, id)
}

func TestClient_LastID_BadStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	_, err = c.LastID(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status")
}

func TestClient_LastID_BadJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, time.Second, testLogger())
	require.NoError(t, err)

	_, err = c.LastID(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode response")
}
