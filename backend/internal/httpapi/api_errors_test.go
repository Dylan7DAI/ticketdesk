//go:build integration

package httpapi_test

import "testing"

// I8：401、400、404、422 各一例，覆盖 WF§9 的格式要求。
func TestI8_ErrorFormats(t *testing.T) {
	e := newEnv(t, false)
	p := e.newTicket()

	// 401：缺头、非数字、不存在的用户、非正整数。
	e.wantErr(e.do("", "GET", "/api/tickets", nil), 401, "UNAUTHENTICATED")
	e.wantErr(e.do("abc", "GET", "/api/tickets", nil), 401, "UNAUTHENTICATED")
	e.wantErr(e.do("999", "GET", "/api/tickets", nil), 401, "UNAUTHENTICATED")
	e.wantErr(e.do("0", "GET", "/api/users", nil), 401, "UNAUTHENTICATED")
	e.wantErr(e.do("-1", "POST", "/api/tickets", validTicket), 401, "UNAUTHENTICATED")
	if r := e.do("", "GET", "/api/health", nil); r.Status != 200 {
		t.Fatalf("/api/health 不应要求身份: %d", r.Status)
	}

	// 400：缺标题带字段级错误；JSON 格式错误；超限的请求体。
	bad := map[string]any{"description": "d", "customer_name": "c", "customer_contact": "x", "priority": "high"}
	b := e.wantErr(e.do(uid(sup), "POST", "/api/tickets", bad), 400, "VALIDATION_ERROR")
	if b.Error.Details["title"] == "" {
		t.Fatalf("details.title 应存在: %+v", b.Error.Details)
	}
	e.wantErr(e.do(uid(sup), "POST", "/api/tickets", `{"title":`), 400, "VALIDATION_ERROR")
	huge := `{"title":"` + string(make([]byte, 70<<10)) + `"}`
	e.wantErr(e.do(uid(sup), "POST", "/api/tickets", huge), 400, "VALIDATION_ERROR")

	// 400：路径 ID 非正整数；列表参数非法。
	e.wantErr(e.do(uid(sup), "GET", "/api/tickets/abc", nil), 400, "VALIDATION_ERROR")
	e.wantErr(e.do(uid(sup), "POST", "/api/tickets/0/claim", nil), 400, "VALIDATION_ERROR")
	e.wantErr(e.do(uid(sup), "GET", "/api/tickets?status=bogus", nil), 400, "VALIDATION_ERROR")
	e.wantErr(e.do(uid(sup), "GET", "/api/tickets?page_size=101", nil), 400, "VALIDATION_ERROR")

	// 404：合法 ID 但不存在，读写都一样。
	e.wantErr(e.do(uid(sup), "GET", "/api/tickets/9999", nil), 404, "TICKET_NOT_FOUND")
	e.wantErr(e.do(uid(lilei), "POST", "/api/tickets/9999/claim", nil), 404, "TICKET_NOT_FOUND")

	// 422：assign 给 supervisor 或不存在的用户。
	e.wantErr(e.do(uid(sup), "POST", p+"/assign", map[string]any{"assignee_id": sup}), 422, "INVALID_ASSIGNEE")
	e.wantErr(e.do(uid(sup), "POST", p+"/assign", map[string]any{"assignee_id": 999}), 422, "INVALID_ASSIGNEE")
	e.wantErr(e.do(uid(sup), "POST", p+"/assign", map[string]any{}), 400, "VALIDATION_ERROR")

	// 409 SAME_ASSIGNEE。
	e.do(uid(sup), "POST", p+"/assign", map[string]any{"assignee_id": lilei})
	e.wantErr(e.do(uid(sup), "POST", p+"/reassign", map[string]any{"assignee_id": lilei}), 409, "SAME_ASSIGNEE")
}
