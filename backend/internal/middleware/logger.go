package middleware

import (
    "bufio"
    "errors"
    "log/slog"
    "net"
    "net/http"
    "time"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// Hijack lets WebSocket upgrades take over the connection through this wrapper.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    h, ok := rw.ResponseWriter.(http.Hijacker)
    if !ok {
        return nil, nil, errors.New("underlying ResponseWriter does not support hijacking")
    }
    rw.statusCode = http.StatusSwitchingProtocols
    return h.Hijack()
}

// Flush passes through streaming flushes.
func (rw *responseWriter) Flush() {
    if f, ok := rw.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}

// Unwrap exposes the underlying writer to http.ResponseController.
func (rw *responseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

// Logger returns a middleware that logs incoming HTTP requests using slog.
func Logger() func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

            next.ServeHTTP(wrapped, r)

            duration := time.Since(start)
            slog.Info("HTTP Request",
                "method", r.Method,
                "path", r.URL.Path,
                "remote_addr", r.RemoteAddr,
                "status", wrapped.statusCode,
                "duration_ms", duration.Milliseconds(),
            )
        })
    }
}