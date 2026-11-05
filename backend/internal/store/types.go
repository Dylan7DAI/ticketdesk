package store

import (
	"encoding/json"
	"time"

	"ticketdesk/internal/domain"
)

// AssigneeFilter 二者互斥：Unassigned=true 对应 assignee_id=none，否则按 UserID 过滤。
type AssigneeFilter struct {
	Unassigned bool
	UserID     int64
}

type TicketFilter struct {
	Statuses   []domain.Status
	Priorities []domain.Priority
	Assignee   *AssigneeFilter // nil 表示不过滤
	Q          string          // 标题或客户名称的子串，大小写不敏感
	Page       int             // 从 1 开始；小于 1 按 1 处理
	PageSize   int             // 小于 1 按 20 处理，超过 100 按 100 处理
	VisibleTo  *int64          // 非 nil 时追加 agent 可见范围条件（WF§11）
}

// TicketRow 是列表中的一行：工单加处理人姓名。
type TicketRow struct {
	domain.Ticket
	AssigneeName *string
}

type Comment struct {
	ID        int64
	TicketID  int64
	AuthorID  int64
	Body      string
	CreatedAt time.Time
}

type Event struct {
	ID        int64
	TicketID  int64
	ActorID   int64
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}
