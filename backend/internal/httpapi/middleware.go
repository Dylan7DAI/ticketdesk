package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"ticketdesk/internal/domain"
)

type ctxKey struct{}

// actor 取出鉴权中间件放入上下文的当前用户。
func actor(r *http.Request) domain.User {
	return r.Context().Value(ctxKey{}).(domain.User)
}

// auth 对 /api/health 以外的 /api/* 要求 X-User-ID（WF§2）。
// 角色只从数据库读取；请求头、请求体里自称的角色一律不看。
func (s *server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		id, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
		if err != nil || id <= 0 {
			writeError(w, r, domain.ErrUnauthenticated())
			return
		}
		u, err := s.svc.Authenticate(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, *u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
