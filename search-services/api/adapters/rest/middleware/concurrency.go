package middleware

import (
	"net/http"
)

func Concurrency(next http.HandlerFunc, limit int) http.HandlerFunc {
	sema := make(chan struct{}, limit)
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case sema <- struct{}{}:
			defer func() { <-sema }()
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "service unavailable (concurrency limit)	", http.StatusServiceUnavailable)
		}
	}
}
