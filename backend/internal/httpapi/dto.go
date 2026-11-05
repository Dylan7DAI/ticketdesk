package httpapi

import (
	"encoding/json"
	"time"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/service"
	"ticketdesk/internal/store"
)

// 本文件的 JSON 形状与 frontend/src/api/types.ts 一一对应（任务分解 2.5 节）。
// 时间用 time.Time，encoding/json 输出 RFC3339。

type userRefDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type userDTO struct {
	ID   int64       `json:"id"`
	Name string      `json:"name"`
	Role domain.Role `json:"role"`
}

type summaryDTO struct {
	ID           int64           `json:"id"`
	Title        string          `json:"title"`
	CustomerName string          `json:"customer_name"`
	Priority     domain.Priority `json:"priority"`
	Status       domain.Status   `json:"status"`
	Assignee     *userRefDTO     `json:"assignee"`
	SLADueAt     time.Time       `json:"sla_due_at"`
	IsOverdue    bool            `json:"is_overdue"`
	CreatedAt    time.Time       `json:"created_at"`
}

type pageDTO struct {
	Items    []summaryDTO `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type commentDTO struct {
	ID        int64      `json:"id"`
	Author    userRefDTO `json:"author"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
}

type eventDTO struct {
	ID        int64           `json:"id"`
	Actor     userRefDTO      `json:"actor"`
	CreatedAt time.Time       `json:"created_at"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Summary   string          `json:"summary"`
}

type detailDTO struct {
	summaryDTO
	Description     string          `json:"description"`
	CustomerContact string          `json:"customer_contact"`
	CreatedBy       userRefDTO      `json:"created_by"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Comments        []commentDTO    `json:"comments"`
	Events          []eventDTO      `json:"events"`
	AllowedActions  []domain.Action `json:"allowed_actions"`
}

func ref(u service.UserRef) userRefDTO { return userRefDTO{u.ID, u.Name} }

func toSummary(s service.TicketSummary) summaryDTO {
	d := summaryDTO{
		ID: s.ID, Title: s.Title, CustomerName: s.CustomerName, Priority: s.Priority,
		Status: s.Status, SLADueAt: s.SLADueAt, IsOverdue: s.IsOverdue, CreatedAt: s.CreatedAt,
	}
	if s.Assignee != nil {
		r := ref(*s.Assignee)
		d.Assignee = &r
	}
	return d
}

func toPage(p *service.TicketPage) pageDTO {
	items := make([]summaryDTO, len(p.Items))
	for i, it := range p.Items {
		items[i] = toSummary(it)
	}
	return pageDTO{items, p.Total, p.Page, p.PageSize}
}

func toDetail(d *service.TicketDetail) detailDTO {
	out := detailDTO{
		summaryDTO: toSummary(d.TicketSummary), Description: d.Description,
		CustomerContact: d.CustomerContact, CreatedBy: ref(d.CreatedBy), UpdatedAt: d.UpdatedAt,
		Comments: make([]commentDTO, len(d.Comments)), Events: make([]eventDTO, len(d.Events)),
		AllowedActions: d.AllowedActions,
	}
	for i, c := range d.Comments {
		out.Comments[i] = commentDTO{c.ID, ref(c.Author), c.Body, c.CreatedAt}
	}
	for i, e := range d.Events {
		out.Events[i] = eventDTO{e.ID, ref(e.Actor), e.CreatedAt, e.Type, e.Payload, e.Summary}
	}
	return out
}

func toComment(c *store.Comment, author service.UserRef) commentDTO {
	return commentDTO{c.ID, ref(author), c.Body, c.CreatedAt}
}
