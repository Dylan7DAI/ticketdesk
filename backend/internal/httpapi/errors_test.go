package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ticketdesk/internal/domain"
)

// WF§9 错误码表。遍历 domain.AllCodes，新增 code 却没映射时这里失败。
func TestEveryCodeIsMapped(t *testing.T) {
	want := map[domain.Code]int{
		domain.CodeValidationError: 400, domain.CodeUnauthenticated: 401,
		domain.CodeForbidden: 403, domain.CodeNotAssignee: 403,
		domain.CodeTicketNotFound: 404, domain.CodeAlreadyClaimed: 409,
		domain.CodeInvalidTransition: 409, domain.CodeInvalidState: 409,
		domain.CodeSameAssignee: 409, domain.CodeInvalidAssignee: 422,
		domain.CodeInternal: 500,
	}
	for _, c := range domain.AllCodes {
		got, ok := statusByCode[c]
		if !ok {
			t.Errorf("code %s 没有 HTTP 状态码映射", c)
		} else if got != want[c] {
			t.Errorf("code %s 应映射到 %d，实际 %d", c, want[c], got)
		}
	}
	if len(statusByCode) != len(domain.AllCodes) {
		t.Errorf("映射表有 %d 项，AllCodes 有 %d 项", len(statusByCode), len(domain.AllCodes))
	}
}

func decodeErr(t *testing.T, rec *httptest.ResponseRecorder) errorDetail {
	t.Helper()
	var b errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("响应不是 WF§9 格式: %v: %s", err, rec.Body)
	}
	return b.Error
}

// 非业务错误返回 500 INTERNAL，响应里不含原始错误文本。
func TestUnknownErrorHidden(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest("GET", "/x", nil), errors.New("pq: password=secret leaked"))
	if rec.Code != 500 {
		t.Fatalf("应为 500，实际 %d", rec.Code)
	}
	e := decodeErr(t, rec)
	if e.Code != domain.CodeInternal || e.Details == nil {
		t.Fatalf("应为 INTERNAL 且 details 为 {}: %+v", e)
	}
	if body := rec.Body.String(); contains(body, "secret") || contains(body, "pq:") {
		t.Fatalf("响应泄露了内部错误: %s", body)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestRecovererReturns500(t *testing.T) {
	h := recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != 500 || decodeErr(t, rec).Code != domain.CodeInternal {
		t.Fatalf("panic 应返回 500 INTERNAL: %d %s", rec.Code, rec.Body)
	}
}
