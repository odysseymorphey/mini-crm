package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (w *responseWriterInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriterInterceptor{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		var userID int64
		if uid, err := GetUserID(r.Context()); err == nil {
			userID = uid
		}

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"bytes_written", rw.written,
		}

		if userID > 0 {
			attrs = append(attrs, "user_id", userID)
		}

		if rw.statusCode >= 500 {
			slog.Error("http request error", attrs...)
		} else if rw.statusCode >= 400 {
			slog.Warn("http client error", attrs...)
		} else {
			slog.Info("http request", attrs...)
		}
	})
}
