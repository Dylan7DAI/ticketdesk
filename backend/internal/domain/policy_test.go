package domain

import (
	"errors"
	"reflect"
	"testing"
)

var (
	sup      = User{ID: 1, Name: "王敏", Role: RoleSupervisor}
	lilei    = User{ID: 2, Name: "李雷", Role: RoleAgent}
	hanmei   = User{ID: 3, Name: "韩梅", Role: RoleAgent}
	zhangwei = User{ID: 4, Name: "张伟", Role: RoleAgent}
)

func i64(v int64) *int64 { return &v }

func ticketWith(status Status, assignee *int64) Ticket {
	return Ticket{ID: 1, Status: status, AssigneeID: assignee}
}

// U1：16 种 (from, to) 组合，恰好 4 个合法。
func TestCanTransition(t *testing.T) {
	all := []Status{StatusOpen, StatusInProgress, StatusResolved, StatusClosed}
	legal := map[[2]Status]bool{
		{StatusOpen, StatusInProgress}:     true,
		{StatusInProgress, StatusResolved}: true,
		{StatusResolved, StatusInProgress}: true,
		{StatusResolved, StatusClosed}:     true,
	}
	count := 0
	for _, from := range all {
		for _, to := range all {
			got := CanTransition(from, to)
			if got != legal[[2]Status{from, to}] {
				t.Errorf("CanTransition(%s, %s) = %v", from, to, got)
			}
			if got {
				count++
			}
		}
	}
	if count != 4 {
		t.Fatalf("合法流转应为 4 条，实际 %d", count)
	}
}

func TestTargetStatus(t *testing.T) {
	want := map[Action]Status{
		ActionResolve: StatusResolved,
		ActionReopen:  StatusInProgress,
		ActionClose:   StatusClosed,
	}
	for a, s := range want {
		if got, ok := TargetStatus(a); !ok || got != s {
			t.Errorf("TargetStatus(%s) = %s, %v", a, got, ok)
		}
	}
	for _, a := range []Action{ActionView, ActionClaim, ActionAssign, ActionReassign, ActionComment} {
		if _, ok := TargetStatus(a); ok {
			t.Errorf("TargetStatus(%s) 不应有目标状态", a)
		}
	}
}

// 判定表单元格：ok 表示允许，其余为错误码缩写（WF§17.1）。
const (
	ok = "ok"
	fb = "F"  // FORBIDDEN
	na = "N"  // NOT_ASSIGNEE
	ac = "AC" // ALREADY_CLAIMED
	it = "IT" // INVALID_TRANSITION
	is = "IS" // INVALID_STATE
)

var codeOf = map[string]Code{
	fb: CodeForbidden,
	na: CodeNotAssignee,
	ac: CodeAlreadyClaimed,
	it: CodeInvalidTransition,
	is: CodeInvalidState,
}

// 列顺序：view, claim, assign, reassign, resolve, reopen, close, comment。
var tableActions = []Action{
	ActionView, ActionClaim, ActionAssign, ActionReassign,
	ActionResolve, ActionReopen, ActionClose, ActionComment,
}

type authRow struct {
	name   string
	actor  User
	ticket Ticket
	want   [8]string
}

// U2：11 种有效的（操作者, 状态）组合 × 8 种动作 = 88 个用例。
// 处理人固定为李雷（ID 2）；"非处理人"取韩梅。
var authTable = []authRow{
	{"主管/open", sup, ticketWith(StatusOpen, nil), [8]string{ok, fb, ok, is, it, it, it, ok}},
	{"主管/in_progress", sup, ticketWith(StatusInProgress, i64(2)), [8]string{ok, fb, ac, ok, ok, it, it, ok}},
	{"主管/resolved", sup, ticketWith(StatusResolved, i64(2)), [8]string{ok, fb, ac, is, it, ok, ok, ok}},
	{"主管/closed", sup, ticketWith(StatusClosed, i64(2)), [8]string{ok, fb, ac, is, it, it, it, is}},
	{"任意agent/open", hanmei, ticketWith(StatusOpen, nil), [8]string{ok, ok, fb, fb, na, na, na, na}},
	{"处理人/in_progress", lilei, ticketWith(StatusInProgress, i64(2)), [8]string{ok, ac, fb, fb, ok, it, it, ok}},
	{"处理人/resolved", lilei, ticketWith(StatusResolved, i64(2)), [8]string{ok, ac, fb, fb, it, ok, ok, ok}},
	{"处理人/closed", lilei, ticketWith(StatusClosed, i64(2)), [8]string{ok, ac, fb, fb, it, it, it, is}},
	{"非处理人/in_progress", hanmei, ticketWith(StatusInProgress, i64(2)), [8]string{ok, ac, fb, fb, na, na, na, na}},
	{"非处理人/resolved", hanmei, ticketWith(StatusResolved, i64(2)), [8]string{ok, ac, fb, fb, na, na, na, na}},
	{"非处理人/closed", hanmei, ticketWith(StatusClosed, i64(2)), [8]string{ok, ac, fb, fb, na, na, na, na}},
}

func TestAuthorize(t *testing.T) {
	cases := 0
	for _, row := range authTable {
		for i, a := range tableActions {
			cases++
			want := row.want[i]
			t.Run(row.name+"/"+string(a), func(t *testing.T) {
				err := Authorize(row.actor, row.ticket, a)
				if want == ok {
					if err != nil {
						t.Fatalf("期望允许，实际 %v", err)
					}
					return
				}
				var de *Error
				if !errors.As(err, &de) {
					t.Fatalf("期望 %s，实际 %v", codeOf[want], err)
				}
				if de.Code != codeOf[want] {
					t.Fatalf("期望 %s，实际 %s", codeOf[want], de.Code)
				}
			})
		}
	}
	if cases != 88 {
		t.Fatalf("U2 应有 88 个用例，实际 %d", cases)
	}
}

// 状态检查按动作的起始状态判断，open 上的 reopen 不能因 open → in_progress 合法而放行。
func TestAuthorize_ReopenOnOpenRejected(t *testing.T) {
	err := Authorize(sup, ticketWith(StatusOpen, nil), ActionReopen)
	var de *Error
	if !errors.As(err, &de) || de.Code != CodeInvalidTransition {
		t.Fatalf("期望 INVALID_TRANSITION，实际 %v", err)
	}
}

func TestAuthorize_ClaimCases(t *testing.T) {
	if err := Authorize(lilei, ticketWith(StatusOpen, nil), ActionClaim); err != nil {
		t.Fatalf("open 上 agent 领取应通过: %v", err)
	}
	err := Authorize(lilei, ticketWith(StatusInProgress, i64(2)), ActionClaim)
	var de *Error
	if !errors.As(err, &de) || de.Code != CodeAlreadyClaimed {
		t.Fatalf("已被自己领取再 claim 应为 ALREADY_CLAIMED，实际 %v", err)
	}
}

// 成功时必须返回真正的 nil，而不是值为 nil 的 *Error。
func TestAuthorize_NoTypedNil(t *testing.T) {
	var err error = Authorize(sup, ticketWith(StatusOpen, nil), ActionAssign)
	if err != nil {
		t.Fatalf("应为 nil: %#v", err)
	}
}

func TestCheckAssignee(t *testing.T) {
	cases := []struct {
		name   string
		ticket Ticket
		target *User
		want   Code // 空串表示通过
	}{
		{"目标不存在", ticketWith(StatusInProgress, i64(2)), nil, CodeInvalidAssignee},
		{"目标是主管", ticketWith(StatusInProgress, i64(2)), &sup, CodeInvalidAssignee},
		{"改派给当前处理人", ticketWith(StatusInProgress, i64(2)), &lilei, CodeSameAssignee},
		{"改派给其他客服", ticketWith(StatusInProgress, i64(2)), &hanmei, ""},
		{"分配时无处理人不触发 SAME_ASSIGNEE", ticketWith(StatusOpen, nil), &lilei, ""},
		{"分配给主管", ticketWith(StatusOpen, nil), &sup, CodeInvalidAssignee},
		{"分配给不存在用户", ticketWith(StatusOpen, nil), nil, CodeInvalidAssignee},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckAssignee(c.ticket, c.target)
			if c.want == "" {
				if err != nil {
					t.Fatalf("应通过: %v", err)
				}
				return
			}
			var de *Error
			if !errors.As(err, &de) || de.Code != c.want {
				t.Fatalf("期望 %s，实际 %v", c.want, err)
			}
		})
	}
}

// AllowedActions：seed 中 6 张工单 × 4 个用户，期望集合与 T5.4 的按钮表一致。
func TestAllowedActions_Seed(t *testing.T) {
	tickets := []struct {
		name string
		t    Ticket
	}{
		{"支付失败无法下单(open)", ticketWith(StatusOpen, nil)},
		{"发票抬头修改(open)", ticketWith(StatusOpen, nil)},
		{"物流长时间未更新(in_progress,李雷)", ticketWith(StatusInProgress, i64(2))},
		{"账号无法登录(in_progress,韩梅)", ticketWith(StatusInProgress, i64(3))},
		{"退款进度查询(resolved,李雷)", ticketWith(StatusResolved, i64(2))},
		{"修改收货地址(closed,张伟)", ticketWith(StatusClosed, i64(4))},
	}
	none := []Action{}
	// want[工单下标][用户]，用户顺序：王敏、李雷、韩梅、张伟。
	want := [6][4][]Action{
		{{ActionAssign, ActionComment}, {ActionClaim}, {ActionClaim}, {ActionClaim}},
		{{ActionAssign, ActionComment}, {ActionClaim}, {ActionClaim}, {ActionClaim}},
		{{ActionReassign, ActionResolve, ActionComment}, {ActionResolve, ActionComment}, none, none},
		{{ActionReassign, ActionResolve, ActionComment}, none, {ActionResolve, ActionComment}, none},
		{{ActionReopen, ActionClose, ActionComment}, {ActionReopen, ActionClose, ActionComment}, none, none},
		{none, none, none, none},
	}
	users := []User{sup, lilei, hanmei, zhangwei}
	for i, tk := range tickets {
		for j, u := range users {
			got := AllowedActions(u, tk.t)
			if got == nil {
				t.Errorf("%s / %s: 应返回空切片而不是 nil", tk.name, u.Name)
			}
			if !reflect.DeepEqual(got, want[i][j]) {
				t.Errorf("%s / %s: 期望 %v，实际 %v", tk.name, u.Name, want[i][j], got)
			}
		}
	}
}

func TestAllowedActions_ExcludesView(t *testing.T) {
	for _, u := range []User{sup, lilei} {
		for _, a := range AllowedActions(u, ticketWith(StatusOpen, nil)) {
			if a == ActionView {
				t.Fatalf("%s: allowed_actions 不应包含 view", u.Name)
			}
		}
	}
}

// 未知角色、未知动作一律按角色检查拒绝，不能落到后面的归属或状态分支。
func TestAuthorize_UnknownRoleAndAction(t *testing.T) {
	ghost := User{ID: 9, Name: "路人", Role: Role("guest")}
	open := ticketWith(StatusOpen, nil)
	for _, a := range tableActions {
		var de *Error
		if err := Authorize(ghost, open, a); !errors.As(err, &de) || de.Code != CodeForbidden {
			t.Errorf("未知角色 %s: 期望 FORBIDDEN，实际 %v", a, err)
		}
	}
	var de *Error
	if err := Authorize(sup, open, Action("delete")); !errors.As(err, &de) || de.Code != CodeForbidden {
		t.Errorf("未知动作: 期望 FORBIDDEN，实际 %v", err)
	}
}
