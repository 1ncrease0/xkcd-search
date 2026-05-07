package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.code = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func WithMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{
			ResponseWriter: w,
			code:           http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		if rw.code == 0 {
			rw.code = http.StatusTeapot
		}

		duration := time.Since(start).Seconds()
		status := http.StatusText(rw.code)
		metrics.GetOrCreatePrometheusHistogram(
			fmt.Sprintf(`http_request_duration_seconds{status=%q,url=%q}`, status, r.URL.Path),
		).Update(duration)
	})
}
