package aaa

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/1ncrease0/xkcd-search/api/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	adminUserEnvKey = "ADMIN_USER"
	adminPassEnvKey = "ADMIN_PASSWORD"
	adminUser       = "admin"
	adminPassword   = "secret"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setAdminEnv(t *testing.T) {
	t.Helper()
	t.Setenv(adminUserEnvKey, adminUser)
	t.Setenv(adminPassEnvKey, adminPassword)
}

func TestNew_MissingAdminUser(t *testing.T) {
	t.Setenv(adminPassEnvKey, adminPassword)

	_, err := New(time.Minute, testLogger())
	require.Error(t, err)
}

func TestNew_MissingAdminPassword(t *testing.T) {
	t.Setenv(adminUserEnvKey, adminUser)

	_, err := New(time.Minute, testLogger())
	require.Error(t, err)
}

func TestNew_Success(t *testing.T) {
	setAdminEnv(t)

	a, err := New(time.Minute, testLogger())
	require.NoError(t, err)
	require.NotEmpty(t, a.users)
}

func TestLogin_Unauthorized(t *testing.T) {
	setAdminEnv(t)

	a, err := New(time.Minute, testLogger())
	require.NoError(t, err)

	_, err = a.Login(adminUser, "wrong")
	require.ErrorIs(t, err, core.ErrUnauthorized)

	_, err = a.Login("other", adminPassword)
	require.ErrorIs(t, err, core.ErrUnauthorized)
}

func TestLogin_AndVerify_Success(t *testing.T) {
	setAdminEnv(t)

	a, err := New(time.Minute, testLogger())
	require.NoError(t, err)

	token, err := a.Login(adminUser, adminPassword)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	require.NoError(t, a.Verify(token))
}

func TestVerify_Expired(t *testing.T) {
	setAdminEnv(t)

	a, err := New(-time.Hour, testLogger())
	require.NoError(t, err)

	token, err := a.Login(adminUser, adminPassword)
	require.NoError(t, err)
	require.Error(t, a.Verify(token))
}

func TestVerify_InvalidTokenString(t *testing.T) {
	t.Parallel()
	var a AAA
	err := a.Verify("not-a-jwt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid token")
}

func TestVerify_WrongSigningKey(t *testing.T) {
	t.Parallel()
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   adminRole,
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString([]byte("wrong-secret"))
	require.NoError(t, err)

	var a AAA
	require.Error(t, a.Verify(s))
}

func TestVerify_WrongSubject(t *testing.T) {
	t.Parallel()
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   "other",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString([]byte(secretKey))
	require.NoError(t, err)

	var a AAA
	require.Error(t, a.Verify(s))
}
