package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"
)

// RequestIDHeader — заголовок идентификатора запроса.
const RequestIDHeader = "X-Request-ID"

// StatusWriter запоминает статус и размер ответа; Unwrap нужен http.ResponseController.
type StatusWriter struct {
	http.ResponseWriter

	status int
	size   int64
}

// AsStatusWriter возвращает w, если это уже *StatusWriter, иначе оборачивает его: обёртка в цепочке одна.
func AsStatusWriter(w http.ResponseWriter) *StatusWriter {
	if sw, ok := w.(*StatusWriter); ok {
		return sw
	}

	return &StatusWriter{ResponseWriter: w}
}

// Status — записанный статус; 200, если хендлер ничего не писал.
func (s *StatusWriter) Status() int {
	if s.status == 0 {
		return http.StatusOK
	}

	return s.status
}

// Size — число записанных байт тела.
func (s *StatusWriter) Size() int64 { return s.size }

// Committed — начат ли ответ.
func (s *StatusWriter) Committed() bool { return s.status != 0 }

func (s *StatusWriter) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *StatusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.size += int64(n)

	return n, err
}

func (s *StatusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// RequestID берёт X-Request-Id запроса или заводит новый и отдаёт его в ответе.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = newRequestID()
			r.Header.Set(RequestIDHeader, id)
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	return hex.EncodeToString(b[:])
}

// Recover превращает панику в ответ fail, если ответ ещё не начат; onPanic получает значение и стек.
// http.ErrAbortHandler пробрасывается дальше: так net/http обрывает соединение.
func Recover(
	onPanic func(r *http.Request, v any, stack []byte),
	fail func(w http.ResponseWriter, r *http.Request),
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := AsStatusWriter(w)
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				onPanic(r, v, debug.Stack())
				if !sw.Committed() {
					fail(sw, r)
				}
			}()
			next.ServeHTTP(sw, r)
		})
	}
}

// Timeout ограничивает контекст запроса; запросы, для которых skip вернул true, идут без срока.
func Timeout(d time.Duration, skip func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip != nil && skip(r) {
				next.ServeHTTP(w, r)

				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLogger пишет одну строку на запрос; уровень — по классу статуса.
func RequestLogger(logger *slog.Logger, clientIP func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := AsStatusWriter(w)
			next.ServeHTTP(sw, r)

			status := sw.Status()
			args := []any{
				slog.String("request_id", r.Header.Get(RequestIDHeader)),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", clientIP(r)),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
				slog.Int64("bytes_in", r.ContentLength),
				slog.Int64("bytes_out", sw.Size()),
			}
			switch {
			case status >= http.StatusInternalServerError:
				logger.ErrorContext(r.Context(), "HTTP request failed", args...)
			case status >= http.StatusBadRequest:
				logger.WarnContext(r.Context(), "HTTP request rejected", args...)
			default:
				logger.InfoContext(r.Context(), "HTTP request completed", args...)
			}
		})
	}
}
