//go:build integration

package httpapi_test

import "testing"

// toState 把新建的 open 工单推进到目标状态，全程以主管身份，排除权限干扰。
func (e *env) toState(state string) string {
	p := e.newTicket()
	steps := map[string][]string{
		"open":        {},
		"in_progress": {"assign"},
		"resolved":    {"assign", "resolve"},
		"closed":      {"assign", "resolve", "close"},
	}[state]
	for _, s := range steps {
		var body any
		if s == "assign" {
			body = map[string]any{"assignee_id": lilei}
		}
		if r := e.do(uid(sup), "POST", p+"/"+s, body); r.Status != 200 {
			e.t.Fatalf("推进到 %s 失败于 %s: %d %s", state, s, r.Status, r.Body)
		}
	}
	return p
}

// I4：非法流转经 HTTP 返回 409，具体码以 WF§17.1 判定表为准。
func TestI4_IllegalTransitions(t *testing.T) {
	e := newEnv(t, false)
	cases := []struct {
		state, action, code string
		status              int
	}{
		{"open", "resolve", "INVALID_TRANSITION", 409},
		{"in_progress", "close", "INVALID_TRANSITION", 409},
		{"closed", "reopen", "INVALID_TRANSITION", 409},
		{"open", "reopen", "INVALID_TRANSITION", 409},
		{"in_progress", "reopen", "INVALID_TRANSITION", 409},
		{"resolved", "resolve", "INVALID_TRANSITION", 409},
		{"closed", "assign", "ALREADY_CLAIMED", 409},
		{"in_progress", "assign", "ALREADY_CLAIMED", 409},
		{"closed", "reassign", "INVALID_STATE", 409},
		{"resolved", "reassign", "INVALID_STATE", 409},
		{"open", "reassign", "INVALID_STATE", 409},
		{"closed", "comments", "INVALID_STATE", 409},
	}
	for _, c := range cases {
		p := e.toState(c.state)
		var body any
		switch c.action {
		case "assign", "reassign":
			body = map[string]any{"assignee_id": hanmei}
		case "comments":
			body = map[string]any{"body": "x"}
		}
		r := e.do(uid(sup), "POST", p+"/"+c.action, body)
		t.Run(c.state+"/"+c.action, func(t *testing.T) { e.wantErr(r, c.status, c.code) })
	}
}

// I5：权限。
func TestI5_Permissions(t *testing.T) {
	e := newEnv(t, false)
	open := e.toState("open")
	mine := e.toState("in_progress") // 处理人是李雷
	assign := map[string]any{"assignee_id": hanmei}

	e.wantErr(e.do(uid(lilei), "POST", open+"/assign", assign), 403, "FORBIDDEN")
	e.wantErr(e.do(uid(lilei), "POST", mine+"/reassign", assign), 403, "FORBIDDEN")
	e.wantErr(e.do(uid(sup), "POST", open+"/claim", nil), 403, "FORBIDDEN")
	e.wantErr(e.do(uid(hanmei), "POST", mine+"/resolve", nil), 403, "NOT_ASSIGNEE")
	e.wantErr(e.do(uid(hanmei), "POST", mine+"/comments", map[string]any{"body": "x"}), 403, "NOT_ASSIGNEE")
	// 详情对所有客服只读可见：能看到，但 allowed_actions 为空。
	r := e.do(uid(hanmei), "GET", mine, nil)
	if r.Status != 200 {
		t.Fatalf("客服应能只读查看他人工单: %d %s", r.Status, r.Body)
	}
	if d := decode[detail](t, r); len(d.AllowedActions) != 0 || d.Assignee == nil || d.Assignee.ID != lilei {
		t.Fatalf("应看到处理人且无任何动作: %+v", d)
	}

	if r := e.do(uid(lilei), "POST", mine+"/resolve", nil); r.Status != 200 {
		t.Fatalf("处理人应能 resolve: %d %s", r.Status, r.Body)
	}
	if r := e.do(uid(hanmei), "POST", open+"/claim", nil); r.Status != 200 {
		t.Fatalf("agent 应能领取 open 工单: %d %s", r.Status, r.Body)
	}
}

// I6：角色来自数据库，请求体和请求头自称的角色无效。
func TestI6_RoleFromDatabase(t *testing.T) {
	e := newEnv(t, false)
	p := e.newTicket()
	body := `{"assignee_id":3,"role":"supervisor","actor_id":1}`
	e.wantErr(e.do(uid(lilei), "POST", p+"/assign", body), 403, "FORBIDDEN")
	e.wantErr(e.do(uid(lilei), "POST", p+"/assign", map[string]any{"assignee_id": hanmei},
		"X-User-Role", "supervisor"), 403, "FORBIDDEN")

	r := e.do(uid(lilei), "GET", "/api/me", nil, "X-User-Role", "supervisor")
	if me := decode[struct{ Role string }](t, r); me.Role != "agent" {
		t.Fatalf("/api/me 应返回数据库中的角色 agent，实际 %s", me.Role)
	}
}
