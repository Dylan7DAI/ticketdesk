package service

import (
	"context"
	"encoding/json"
	"unicode/utf8"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// 事件类型，与 ticket_events.type 的 CHECK 约束一致（WF§10）。
const (
	EventCreated       = "created"
	EventClaimed       = "claimed"
	EventAssigned      = "assigned"
	EventReassigned    = "reassigned"
	EventStatusChanged = "status_changed"
	EventCommented     = "commented"
)

// 各事件的 payload（WF§6）。用结构体加 json.Marshal 构造，不手写 JSON 字符串。
type (
	createdPayload struct {
		Priority domain.Priority `json:"priority"`
	}
	assigneePayload struct {
		AssigneeID int64 `json:"assignee_id"`
	}
	reassignedPayload struct {
		From int64 `json:"from_assignee_id"`
		To   int64 `json:"to_assignee_id"`
	}
	statusPayload struct {
		From domain.Status `json:"from"`
		To   domain.Status `json:"to"`
		Note string        `json:"note"`
	}
	commentedPayload struct {
		CommentID int64  `json:"comment_id"`
		Excerpt   string `json:"excerpt"`
	}
)

// excerptLen 是评论摘要的字符数上限（WF§6：前 100 字符）。
const excerptLen = 100

func excerpt(s string) string {
	if utf8.RuneCountInString(s) <= excerptLen {
		return s
	}
	return string([]rune(s)[:excerptLen])
}

// addEvent 在事务内写入一条事件；payload 序列化失败时返回错误，使整个事务回滚。
func addEvent(ctx context.Context, q store.Querier, ticketID, actorID int64, typ string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.InsertEvent(ctx, &store.Event{
		TicketID: ticketID, ActorID: actorID, Type: typ, Payload: raw,
	})
}
