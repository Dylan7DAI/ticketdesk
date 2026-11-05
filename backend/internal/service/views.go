package service

import (
	"strconv"
	"time"

	"ticketdesk/internal/domain"
)

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

// UserRef 对应前端的 UserRef。
type UserRef struct {
	ID   int64
	Name string
}

// TicketSummary 对应前端的 TicketSummary；IsOverdue 由 service 用注入时钟计算。
type TicketSummary struct {
	ID           int64
	Title        string
	CustomerName string
	Priority     domain.Priority
	Status       domain.Status
	Assignee     *UserRef
	SLADueAt     time.Time
	IsOverdue    bool
	CreatedAt    time.Time
}

type TicketPage struct {
	Items    []TicketSummary
	Total    int
	Page     int
	PageSize int
}

type CommentView struct {
	ID        int64
	Author    UserRef
	Body      string
	CreatedAt time.Time
}

type EventView struct {
	ID        int64
	Actor     UserRef
	Type      string
	Payload   []byte
	Summary   string
	CreatedAt time.Time
}

// TicketDetail 对应前端的 TicketDetail。
type TicketDetail struct {
	TicketSummary
	Description     string
	CustomerContact string
	CreatedBy       UserRef
	UpdatedAt       time.Time
	Comments        []CommentView
	Events          []EventView
	AllowedActions  []domain.Action
}

func userRef(n names, id int64) UserRef { return UserRef{ID: id, Name: n.raw(id)} }

// raw 返回完整名称（含角色前缀），用于 UserRef；摘要里才用 shortName。
func (n names) raw(id int64) string {
	if v, ok := n[id]; ok {
		return v
	}
	return "用户 #" + itoa(id)
}

func summarize(t domain.Ticket, assignee *UserRef, now time.Time) TicketSummary {
	return TicketSummary{
		ID: t.ID, Title: t.Title, CustomerName: t.CustomerName,
		Priority: t.Priority, Status: t.Status, Assignee: assignee,
		SLADueAt: t.SLADueAt, IsOverdue: domain.IsOverdue(t, now), CreatedAt: t.CreatedAt,
	}
}
