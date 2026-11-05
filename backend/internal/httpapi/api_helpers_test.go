//go:build integration

package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ticketdesk/internal/httpapi"
	"ticketdesk/internal/service"
	"ticketdesk/internal/testutil"
)

// 与 testutil 插入的固定用户一致（WF§1）。
const (
	sup    = 1
	lilei  = 2
	hanmei = 3
)

type env struct {
	t   *testing.T
	url string
}

type resp struct {
	Status int
	Body   []byte
}

func newEnv(t *testing.T, seeded bool) *env {
	t.Helper()
	s := testutil.NewTestStore(t)
	if seeded {
		s = testutil.NewSeededStore(t)
	}
	srv := httptest.NewServer(httpapi.NewServer(service.New(s, nil)))
	t.Cleanup(srv.Close)
	return &env{t: t, url: srv.URL}
}

// do 以 userID 身份发请求；userID 为空串则不带 X-User-ID。extra 是额外的请求头键值对。
func (e *env) do(userID, method, path string, body any, extra ...string) resp {
	e.t.Helper()
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case string:
		rd = bytes.NewReader([]byte(b))
	default:
		raw, _ := json.Marshal(b)
		rd = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, e.url+path, rd)
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
	for i := 0; i+1 < len(extra); i += 2 {
		req.Header.Set(extra[i], extra[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(res.Body)
	return resp{res.StatusCode, buf.Bytes()}
}

func uid(id int) string { return strconv.Itoa(id) }

type errResp struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Details map[string]string `json:"details"`
	} `json:"error"`
}

// wantErr 断言状态码和 WF§9 错误码，并返回错误体。
func (e *env) wantErr(r resp, status int, code string) errResp {
	e.t.Helper()
	var b errResp
	if err := json.Unmarshal(r.Body, &b); err != nil {
		e.t.Fatalf("响应不是 WF§9 格式: %v: %s", err, r.Body)
	}
	if r.Status != status || b.Error.Code != code {
		e.t.Fatalf("期望 %d %s，实际 %d %s (%s)", status, code, r.Status, b.Error.Code, b.Error.Message)
	}
	if b.Error.Details == nil {
		e.t.Fatalf("details 应为 {} 而不是 null: %s", r.Body)
	}
	return b
}

func decode[T any](t *testing.T, r resp) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("解码失败: %v: %s", err, r.Body)
	}
	return v
}

var validTicket = map[string]any{
	"title": "支付失败", "description": "无法下单", "customer_name": "客户甲",
	"customer_contact": "a@b.c", "priority": "high",
}

// newTicket 以主管身份创建一张 open 工单，返回其 ID 路径前缀。
func (e *env) newTicket() string {
	e.t.Helper()
	r := e.do(uid(sup), "POST", "/api/tickets", validTicket)
	if r.Status != 201 {
		e.t.Fatalf("创建失败: %d %s", r.Status, r.Body)
	}
	d := decode[struct {
		ID int `json:"id"`
	}](e.t, r)
	return "/api/tickets/" + strconv.Itoa(d.ID)
}
