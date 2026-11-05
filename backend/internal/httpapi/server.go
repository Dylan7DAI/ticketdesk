// Package httpapi 是 HTTP 外壳：路由、鉴权中间件、请求解析和 domain 错误码 → HTTP 状态码。
// 这里不做业务判断，也不直接调用 store。
package httpapi

import (
	"net/http"

	"ticketdesk/internal/service"
)

type server struct{ svc *service.Service }

// NewServer 返回挂好全部路由和中间件的 Handler。
// 中间件顺序（外 → 内）：请求日志 → panic 恢复 → 鉴权 → 路由。
func NewServer(svc *service.Service) http.Handler {
	s := &server{svc: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", health)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("GET /api/users", s.listUsers)
	mux.HandleFunc("GET /api/tickets", s.listTickets)
	mux.HandleFunc("POST /api/tickets", s.createTicket)
	mux.HandleFunc("GET /api/tickets/{id}", s.getTicket)
	mux.HandleFunc("POST /api/tickets/{id}/claim", s.claim)
	mux.HandleFunc("POST /api/tickets/{id}/assign", s.assign)
	mux.HandleFunc("POST /api/tickets/{id}/reassign", s.reassign)
	mux.HandleFunc("POST /api/tickets/{id}/resolve", s.transition("resolve"))
	mux.HandleFunc("POST /api/tickets/{id}/reopen", s.transition("reopen"))
	mux.HandleFunc("POST /api/tickets/{id}/close", s.transition("close"))
	mux.HandleFunc("POST /api/tickets/{id}/comments", s.addComment)

	return logging(recoverer(s.auth(mux)))
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
