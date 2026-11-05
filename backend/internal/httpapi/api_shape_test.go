//go:build integration

package httpapi_test

import (
	"encoding/json"
	"testing"
	"time"
)

type ref struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// 与 frontend/src/api/types.ts 的 TicketDetail 逐字段对应。
type detail struct {
	ID              int             `json:"id"`
	Title           string          `json:"title"`
	CustomerName    string          `json:"customer_name"`
	Priority        string          `json:"priority"`
	Status          string          `json:"status"`
	Assignee        *ref            `json:"assignee"`
	SLADueAt        time.Time       `json:"sla_due_at"`
	IsOverdue       bool            `json:"is_overdue"`
	CreatedAt       time.Time       `json:"created_at"`
	Description     string          `json:"description"`
	CustomerContact string          `json:"customer_contact"`
	CreatedBy       ref             `json:"created_by"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Comments        []comment       `json:"comments"`
	Events          []event         `json:"events"`
	AllowedActions  []string        `json:"allowed_actions"`
	Extra           json.RawMessage `json:"-"`
}

type comment struct {
	ID        int       `json:"id"`
	Author    ref       `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type event struct {
	ID        int            `json:"id"`
	Actor     ref            `json:"actor"`
	CreatedAt time.Time      `json:"created_at"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	Summary   string         `json:"summary"`
}

func checkDetail(t *testing.T, name string, r resp, wantStatus int, wantState string) detail {
	t.Helper()
	if r.Status != wantStatus {
		t.Fatalf("%s: 期望 %d，实际 %d %s", name, wantStatus, r.Status, r.Body)
	}
	d := decode[detail](t, r)
	if d.ID == 0 || d.Title == "" || d.CreatedBy.ID == 0 || d.CreatedBy.Name == "" ||
		d.SLADueAt.IsZero() || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() ||
		len(d.Events) == 0 || d.Events[0].Summary == "" || d.Events[0].Actor.Name == "" {
		t.Fatalf("%s: 关键字段为零值: %s", name, r.Body)
	}
	if d.Status != wantState {
		t.Fatalf("%s: 状态应为 %s，实际 %s", name, wantState, d.Status)
	}
	if d.Comments == nil || d.AllowedActions == nil || d.Events == nil {
		t.Fatalf("%s: 数组字段必须是 [] 而不是 null: %s", name, r.Body)
	}
	return d
}

func TestWriteResponsesMatchTypes(t *testing.T) {
	e := newEnv(t, false)
	r := e.do(uid(sup), "POST", "/api/tickets", validTicket)
	d := checkDetail(t, "create", r, 201, "open")
	p := "/api/tickets/" + itoa(d.ID)
	if d.Assignee != nil || len(d.Events) != 1 || d.Events[0].Type != "created" {
		t.Fatalf("新工单无处理人、仅 created 事件: %+v", d)
	}

	d = checkDetail(t, "claim", e.do(uid(lilei), "POST", p+"/claim", nil), 200, "in_progress")
	if d.Assignee == nil || d.Assignee.ID != lilei || d.Assignee.Name == "" {
		t.Fatalf("claim 后应带处理人: %+v", d.Assignee)
	}
	checkDetail(t, "reassign", e.do(uid(sup), "POST", p+"/reassign", map[string]any{"assignee_id": hanmei}), 200, "in_progress")
	checkDetail(t, "resolve", e.do(uid(hanmei), "POST", p+"/resolve", map[string]any{"note": "ok"}), 200, "resolved")
	checkDetail(t, "reopen", e.do(uid(hanmei), "POST", p+"/reopen", nil), 200, "in_progress")
	checkDetail(t, "resolve2", e.do(uid(sup), "POST", p+"/resolve", nil), 200, "resolved")
	d = checkDetail(t, "close", e.do(uid(sup), "POST", p+"/close", nil), 200, "closed")
	if len(d.AllowedActions) != 0 {
		t.Fatalf("closed 不应有任何动作: %v", d.AllowedActions)
	}

	// assign 单独验证（claim 已占用上面的工单）。
	p2 := e.newTicket()
	checkDetail(t, "assign", e.do(uid(sup), "POST", p2+"/assign", map[string]any{"assignee_id": lilei}), 200, "in_progress")

	// 评论返回 201 和新评论（不是工单）。
	c := decode[comment](t, e.do(uid(sup), "POST", p2+"/comments", map[string]any{"body": "跟进"}))
	if c.ID == 0 || c.Body != "跟进" || c.Author.ID != sup || c.Author.Name == "" || c.CreatedAt.IsZero() {
		t.Fatalf("评论响应字段不对: %+v", c)
	}
	if r := e.do(uid(sup), "POST", p2+"/comments", map[string]any{"body": "x"}); r.Status != 201 {
		t.Fatalf("评论应返回 201，实际 %d", r.Status)
	}
}

func itoa(n int) string { return uid(n) }
