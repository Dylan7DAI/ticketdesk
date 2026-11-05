package domain

import "time"

// Role 是用户角色，只从数据库读取（WF§2）。
type Role string

const (
	RoleAgent      Role = "agent"
	RoleSupervisor Role = "supervisor"
)

func (r Role) Valid() bool { return r == RoleAgent || r == RoleSupervisor }

func ParseRole(s string) (Role, bool) {
	r := Role(s)
	return r, r.Valid()
}

// Status 是工单状态（WF§3）。
type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed:
		return true
	}
	return false
}

func ParseStatus(v string) (Status, bool) {
	s := Status(v)
	return s, s.Valid()
}

// Priority 是工单优先级，决定 SLA 时长（WF§8）。
type Priority string

const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityUrgent, PriorityHigh, PriorityNormal, PriorityLow:
		return true
	}
	return false
}

func ParsePriority(v string) (Priority, bool) {
	p := Priority(v)
	return p, p.Valid()
}

// Action 是对工单可执行的操作（WF§4、§6）。view 只用于权限判定，不出现在 allowed_actions 中。
type Action string

const (
	ActionView     Action = "view"
	ActionClaim    Action = "claim"
	ActionAssign   Action = "assign"
	ActionReassign Action = "reassign"
	ActionResolve  Action = "resolve"
	ActionReopen   Action = "reopen"
	ActionClose    Action = "close"
	ActionComment  Action = "comment"
)

// ActionsForUI 是 AllowedActions 依次判定的动作，顺序即返回顺序。
var ActionsForUI = []Action{
	ActionClaim, ActionAssign, ActionReassign,
	ActionResolve, ActionReopen, ActionClose, ActionComment,
}

func (a Action) Valid() bool {
	if a == ActionView {
		return true
	}
	for _, x := range ActionsForUI {
		if a == x {
			return true
		}
	}
	return false
}

func ParseAction(v string) (Action, bool) {
	a := Action(v)
	return a, a.Valid()
}

type User struct {
	ID   int64
	Name string
	Role Role
}

type Ticket struct {
	ID              int64
	Title           string
	Description     string
	CustomerName    string
	CustomerContact string
	Priority        Priority
	Status          Status
	AssigneeID      *int64
	CreatedBy       int64
	SLADueAt        time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
