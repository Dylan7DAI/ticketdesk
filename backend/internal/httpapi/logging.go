package httpapi

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"ticketdesk/internal/domain"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// logging 记录方法、路径、用户 ID、状态码和耗时。它在最外层，所以 panic 恢复后的 500 也会被记录。
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s user=%q status=%d %s",
			r.Method, r.URL.Path, r.Header.Get("X-User-ID"), rec.status, time.Since(start))
	})
}

// recoverer 捕获 handler 的 panic，记录堆栈并返回 500 INTERNAL。
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("panic: %v\n%s", p, debug.Stack())
				writeError(w, r, domain.ErrInternal())
			}
		}()
		next.ServeHTTP(w, r)
	})
}
