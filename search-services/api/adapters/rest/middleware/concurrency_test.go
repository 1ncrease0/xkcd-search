package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConcurrency_OK(t *testing.T) {
	t.Parallel()
	called := false
	h := Concurrency(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}, 2)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, called)
}

func TestConcurrency_503(t *testing.T) {
	proceed := make(chan struct{})
	started := make(chan struct{})

	h := Concurrency(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-proceed
		w.WriteHeader(http.StatusOK)
	}, 1)

	go func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/first", nil))
	}()

	<-started

	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/second", nil))
	require.Equal(t, http.StatusServiceUnavailable, rr2.Code)
	require.Contains(t, rr2.Body.String(), "concurrency limit")

	close(proceed)
	time.Sleep(20 * time.Millisecond)
}
