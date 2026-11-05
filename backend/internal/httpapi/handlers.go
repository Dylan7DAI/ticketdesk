package httpapi

import (
	"net/http"

	"ticketdesk/internal/domain"
)

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	u := actor(r)
	writeJSON(w, http.StatusOK, userDTO{u.ID, u.Name, u.Role})
}

func (s *server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.svc.ListUsers(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]userDTO, len(users))
	for i, u := range users {
		out[i] = userDTO{u.ID, u.Name, u.Role}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) listTickets(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r.URL.Query())
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := s.svc.ListTickets(r.Context(), actor(r), f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPage(page))
}

func (s *server) getTicket(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	d, err := s.svc.GetTicketDetail(r.Context(), actor(r), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDetail(d))
}

type createReq struct {
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	CustomerName    string          `json:"customer_name"`
	CustomerContact string          `json:"customer_contact"`
	Priority        domain.Priority `json:"priority"`
}

func (s *server) createTicket(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	a := actor(r)
	t, err := s.svc.CreateTicket(r.Context(), a, domain.CreateTicketInput{
		Title: req.Title, Description: req.Description, CustomerName: req.CustomerName,
		CustomerContact: req.CustomerContact, Priority: req.Priority,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	s.respondDetail(w, r, http.StatusCreated, a, t)
}

// respondDetail 在写操作成功后返回最新的 TicketDetail（WF§17.2）。
func (s *server) respondDetail(w http.ResponseWriter, r *http.Request, status int, a domain.User, t *domain.Ticket) {
	d, err := s.svc.Detail(r.Context(), a, t)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, status, toDetail(d))
}

type assignReq struct {
	AssigneeID int64 `json:"assignee_id"`
}

type noteReq struct {
	Note string `json:"note"`
}

type commentReq struct {
	Body string `json:"body"`
}
