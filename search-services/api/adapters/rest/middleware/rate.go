package middleware

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

func Rate(next http.HandlerFunc, rps int) http.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), 1)
	return func(w http.ResponseWriter, r *http.Request) {
		err := limiter.Wait(r.Context())
		if err != nil {
			fmt.Println(err)
			http.Error(w, "server is going down", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	}
}
