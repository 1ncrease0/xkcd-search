package aaa

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/1ncrease0/xkcd-search/api/core"
)

const secretKey = "something secret here"
const adminRole = "superuser"

type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	log      *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger) (AAA, error) {
	const adminUser = "ADMIN_USER"
	const adminPass = "ADMIN_PASSWORD"
	user, ok := os.LookupEnv(adminUser)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin user from enviroment")
	}
	password, ok := os.LookupEnv(adminPass)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin password from enviroment")
	}

	return AAA{
		users:    map[string]string{user: password},
		tokenTTL: tokenTTL,
		log:      log,
	}, nil
}

func (a AAA) Login(name, password string) (string, error) {
	expected, ok := a.users[name]
	if !ok || expected != password {
		return "", core.ErrUnauthorized
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   adminRole,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.tokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secretKey))
	if err != nil {
		a.log.Error("could not sign JWT", "error", err)
		return "", fmt.Errorf("could not sign token: %w", err)
	}

	return signed, nil
}

func (a AAA) Verify(tokenString string) error {
	_, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			return []byte(secretKey), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithExpirationRequired(),
		jwt.WithSubject(adminRole),
	)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	return nil
}
