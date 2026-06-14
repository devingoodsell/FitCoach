package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey int

const requestIDKey ctxKey = iota

// RequestIDHeader is the header carrying the correlation ID, both inbound
// (honored if a trusted proxy set it) and outbound (always echoed).
const RequestIDHeader = "X-Request-ID"

// RequestIDFromContext returns the request ID stored by Middleware, or "".
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// statusRecorder captures the status code for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware assigns a request ID, stores it in the context, echoes it in the
// response header, and logs method/path/status/latency/request-id for every
// request. The logger's redaction hook keeps sensitive attributes out.
func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if id == "" {
				id = newRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			w.Header().Set(RequestIDHeader, id)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r.WithContext(ctx))

			logger.LogAttrs(ctx, slog.LevelInfo, "http_request",
				slog.String("request_id", id),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("latency", time.Since(start)),
			)
		})
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read failing is effectively impossible; fall back to a timestamp.
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
