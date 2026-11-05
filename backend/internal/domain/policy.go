package domain

// 合法流转只有 4 条（WF§3），其余全部拒绝。
var legalTransitions = map[Status]map[Status]bool{
	StatusOpen:       {StatusInProgress: true},
	StatusInProgress: {StatusResolved: true},
	StatusResolved:   {StatusInProgress: true, StatusClosed: true},
}

// CanTransition 判断 from → to 是否是 4 条合法流转之一。
func CanTransition(from, to Status) bool {
	return legalTransitions[from][to]
}

// 流转动作的目标状态与固定起始状态（WF§17.1）。
// 状态检查必须按"动作要求的起始状态"判断：reopen 的目标 in_progress 与
// open → in_progress 本身合法，只查 CanTransition 会错误放行 open 上的 reopen。
var (
	targetStatus = map[Action]Status{
		ActionResolve: StatusResolved,
		ActionReopen:  StatusInProgress,
		ActionClose:   StatusClosed,
	}
	requiredFrom = map[Action]Status{
		ActionResolve: StatusInProgress,
		ActionReopen:  StatusResolved,
		ActionClose:   StatusResolved,
	}
)

// TargetStatus 返回流转动作 resolve / reopen / close 的目标状态。
// claim、assign 的目标状态固定为 in_progress，由各自的写操作直接设置，这里不返回。
func TargetStatus(a Action) (Status, bool) {
	s, ok := targetStatus[a]
	return s, ok
}

// Authorize 按 WF§5 第 5～7 步的顺序（角色 → 归属 → 状态）返回第一个错误。
// 成功时返回 nil；失败时返回 *Error。完整判定表见 WF§17.1。
func Authorize(actor User, t Ticket, a Action) error {
	if !roleAllowed(actor.Role, a) {
		return ErrForbidden()
	}
	if err := checkOwnership(actor, t, a); err != nil {
		return err
	}
	return checkState(t, a)
}

// roleAllowed：claim 只允许 agent，assign / reassign 只允许 supervisor，
// 其余动作两种角色都可以；未知角色或未知动作一律拒绝。
func roleAllowed(r Role, a Action) bool {
	if !r.Valid() {
		return false
	}
	switch a {
	case ActionView, ActionResolve, ActionReopen, ActionClose, ActionComment:
		return true
	case ActionClaim:
		return r == RoleAgent
	case ActionAssign, ActionReassign:
		return r == RoleSupervisor
	}
	return false
}

// unassigned 表示工单当前没有处理人。
func unassigned(t Ticket) bool { return t.AssigneeID == nil }

func isAssignee(actor User, t Ticket) bool {
	return t.AssigneeID != nil && *t.AssigneeID == actor.ID
}

// checkOwnership：supervisor 不受归属限制；agent 的 claim 和 view 始终豁免
// （详情对所有客服只读可见，WF§4），其余动作要求是当前处理人。
func checkOwnership(actor User, t Ticket, a Action) error {
	if actor.Role == RoleSupervisor || a == ActionClaim || a == ActionView {
		return nil
	}
	if isAssignee(actor, t) {
		return nil
	}
	return ErrNotAssignee()
}

// checkState 检查动作要求的起始状态和前置条件（WF§3、§6）。
func checkState(t Ticket, a Action) error {
	switch a {
	case ActionClaim, ActionAssign:
		if t.Status != StatusOpen || !unassigned(t) {
			return ErrAlreadyClaimed()
		}
	case ActionReassign:
		if t.Status != StatusInProgress {
			return ErrInvalidState()
		}
	case ActionResolve, ActionReopen, ActionClose:
		if t.Status != requiredFrom[a] {
			return ErrInvalidTransition()
		}
	case ActionComment:
		if t.Status == StatusClosed {
			return ErrInvalidState()
		}
	}
	return nil
}

// CheckAssignee 是 WF§5 第 8 步的目标处理人检查。target 为 nil 表示用户不存在。
// 目标须是 agent；改派时目标不能等于当前处理人。工单无处理人（assign）时不会触发 SAME_ASSIGNEE。
func CheckAssignee(t Ticket, target *User) error {
	if target == nil || target.Role != RoleAgent {
		return ErrInvalidAssignee()
	}
	if t.AssigneeID != nil && *t.AssigneeID == target.ID {
		return ErrSameAssignee()
	}
	return nil
}

// AllowedActions 对 claim..comment 逐个调用 Authorize，返回无错误的动作，供详情页渲染按钮。
// view 不在其中：能拿到详情即代表可查看。
func AllowedActions(actor User, t Ticket) []Action {
	out := make([]Action, 0, len(ActionsForUI))
	for _, a := range ActionsForUI {
		if Authorize(actor, t, a) == nil {
			out = append(out, a)
		}
	}
	return out
}
