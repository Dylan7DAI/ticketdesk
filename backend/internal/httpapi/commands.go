package httpapi

import (
	"net/http"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/service"
)

// command 抽出写接口的共同骨架：解析路径 ID → 执行 service 方法 → 返回最新详情。
// 角色、归属、状态的判断全在 service/domain，这里不做任何业务判断。
func (s *server) command(w http.ResponseWriter, r *http.Request, run func(a domain.User, id int64) (*domain.Ticket, error)) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	a := actor(r)
	t, err := run(a, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	s.respondDetail(w, r, http.StatusOK, a, t)
}

func (s *server) claim(w http.ResponseWriter, r *http.Request) {
	s.command(w, r, func(a domain.User, id int64) (*domain.Ticket, error) {
		return s.svc.Claim(r.Context(), a, id)
	})
}

// assignLike 处理 assign 与 reassign：请求体都只有 assignee_id。
func (s *server) assignLike(op func(*service.Service, *http.Request, domain.User, int64, int64) (*domain.Ticket, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req assignReq
		if err := readJSON(w, r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		s.command(w, r, func(a domain.User, id int64) (*domain.Ticket, error) {
			return op(s.svc, r, a, id, req.AssigneeID)
		})
	}
}

func (s *server) assign(w http.ResponseWriter, r *http.Request) {
	s.assignLike(func(v *service.Service, r *http.Request, a domain.User, id, to int64) (*domain.Ticket, error) {
		return v.Assign(r.Context(), a, id, to)
	})(w, r)
}

func (s *server) reassign(w http.ResponseWriter, r *http.Request) {
	s.assignLike(func(v *service.Service, r *http.Request, a domain.User, id, to int64) (*domain.Ticket, error) {
		return v.Reassign(r.Context(), a, id, to)
	})(w, r)
}

// transition 返回 resolve / reopen / close 的 handler，note 可选。
func (s *server) transition(action domain.Action) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req noteReq
		if err := readJSON(w, r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		s.command(w, r, func(a domain.User, id int64) (*domain.Ticket, error) {
			return s.svc.Transition(r.Context(), a, id, action, req.Note)
		})
	}
}

// addComment 返回 201 和新建的评论（不是整张工单）。
func (s *server) addComment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req commentReq
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	a := actor(r)
	c, err := s.svc.AddComment(r.Context(), a, id, req.Body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toComment(c, service.UserRef{ID: a.ID, Name: a.Name}))
}
