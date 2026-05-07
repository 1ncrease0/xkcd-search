package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const validTestToken = "abc.def.ghi"

type stubVerifier struct {
	err error
}

func (s stubVerifier) Verify(string) error {
	return s.err
}

type recordingVerifier struct {
	lastToken string
	err       error
}

func (r *recordingVerifier) Verify(token string) error {
	r.lastToken = token
	return r.err
}

func TestAuth_Unauthorized_MissingHeader(t *testing.T) {
	t.Parallel()

	called := false
	h := Auth(func(http.ResponseWriter, *http.Request) { called = true }, stubVerifier{err: nil})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.False(t, called)
}

func TestAuth_Unauthorized_WrongPrefix(t *testing.T) {
	t.Parallel()

	called := false
	h := Auth(func(http.ResponseWriter, *http.Request) { called = true }, stubVerifier{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer x")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.False(t, called)
}

func TestAuth_Unauthorized_EmptyToken(t *testing.T) {
	t.Parallel()

	called := false
	h := Auth(func(http.ResponseWriter, *http.Request) { called = true }, stubVerifier{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token ")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.False(t, called)
}

func TestAuth_Unauthorized_VerifyFails(t *testing.T) {
	t.Parallel()

	called := false
	h := Auth(func(http.ResponseWriter, *http.Request) { called = true }, stubVerifier{err: http.ErrNoCookie})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token bad")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.False(t, called)
}

func TestAuth_Authorized_ValidToken(t *testing.T) {
	t.Parallel()

	called := false
	ver := &recordingVerifier{err: nil}
	h := Auth(func(http.ResponseWriter, *http.Request) { called = true }, ver)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token "+validTestToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, called)
	require.Equal(t, validTestToken, ver.lastToken)
}

func TestAuth_Unauthorized_ErrorMessage(t *testing.T) {
	t.Parallel()

	h := Auth(func(http.ResponseWriter, *http.Request) {}, stubVerifier{err: nil})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, "unauthorized\n", rr.Body.String())
}
