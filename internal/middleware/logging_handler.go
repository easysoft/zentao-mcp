package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const redactedValue = "[REDACTED]"

var sensitiveQueryKeys = map[string]struct{}{
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"password":      {},
	"passwd":        {},
	"refresh_token": {},
	"secret":        {},
	"token":         {},
	"access_token":  {},
}

// LoggingHandler records one access log entry after each HTTP request.
type LoggingHandler struct {
	handler http.Handler
	logger  *slog.Logger
}

// NewLoggingHandler wraps an http.Handler with access logging.
func NewLoggingHandler(handler http.Handler) http.Handler {
	return &LoggingHandler{
		handler: handler,
		logger:  slog.Default().With("component", "middleware.access"),
	}
}

func (h *LoggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	lrw := &loggingResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}

	h.handler.ServeHTTP(lrw, r)

	h.logger.InfoContext(r.Context(), "request completed",
		"method", r.Method,
		"path", r.URL.Path,
		"query_params", redactQueryParams(r.URL.Query()),
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"status", lrw.status,
		"duration", time.Since(started),
	)
}

func redactQueryParams(values url.Values) map[string][]string {
	redacted := make(map[string][]string, len(values))

	for key, vals := range values {
		copied := append([]string(nil), vals...)
		if _, ok := sensitiveQueryKeys[strings.ToLower(key)]; ok {
			for i := range copied {
				copied[i] = redactedValue
			}
		}

		redacted[key] = copied
	}

	return redacted
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
