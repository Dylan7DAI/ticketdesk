//go:build integration

package service_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// step 断言一步成功后的状态、处理人、事件总数，以及最后一条事件的类型与 payload。
func step(t *testing.T, name string, s *store.Store, id int64, status domain.Status, assignee int64, nEvents int, lastType, lastPayload string) {
	t.Helper()
	got, err := s.Q().GetTicket(ctx, id)
	if err != nil || got == nil {
		t.Fatalf("%s: 读工单失败 %v", name, err)
	}
	if got.Status != status {
		t.Fatalf("%s: 状态应为 %s，实际 %s", name, status, got.Status)
	}
	if assignee == 0 && got.AssigneeID != nil || assignee != 0 && (got.AssigneeID == nil || *got.AssigneeID != assignee) {
		t.Fatalf("%s: 处理人应为 %d，实际 %v", name, assignee, got.AssigneeID)
	}
	evs := events(t, s.Q(), id)
	if len(evs) != nEvents {
		t.Fatalf("%s: 事件应有 %d 条，实际 %d", name, nEvents, len(evs))
	}
	last := evs[len(evs)-1]
	var want, have any
	_ = json.Unmarshal([]byte(lastPayload), &want)
	_ = json.Unmarshal(last.Payload, &have)
	if last.Type != lastType || !jsonEqual(want, have) {
		t.Fatalf("%s: 最后事件应为 %s %s，实际 %s %s", name, lastType, lastPayload, last.Type, last.Payload)
	}
}

func jsonEqual(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// I3：create → claim → comment → resolve → reopen → reassign → resolve → close，
// 中间穿插失败调用，失败后事件数必须不变。
func TestLifecycle(t *testing.T) {
	svc, s := newSvc(t)
	tk := newTicket(t, svc)
	id := tk.ID
	step(t, "create", s, id, domain.StatusOpen, 0, 1, "created", `{"priority":"high"}`)

	_, err := svc.Transition(ctx, lilei, id, domain.ActionResolve, "")
	wantCode(t, err, domain.CodeNotAssignee)
	step(t, "失败后", s, id, domain.StatusOpen, 0, 1, "created", `{"priority":"high"}`)

	if _, err := svc.Claim(ctx, lilei, id); err != nil {
		t.Fatal(err)
	}
	step(t, "claim", s, id, domain.StatusInProgress, 2, 2, "claimed", `{"assignee_id":2}`)

	_, err = svc.AddComment(ctx, hanmei, id, "我也想评论")
	wantCode(t, err, domain.CodeNotAssignee)
	c, err := svc.AddComment(ctx, lilei, id, "已联系客户")
	if err != nil {
		t.Fatal(err)
	}
	step(t, "comment", s, id, domain.StatusInProgress, 2, 3, "commented", `{"comment_id":`+strconv.FormatInt(c.ID, 10)+`,"excerpt":"已联系客户"}`)

	if _, err := svc.Transition(ctx, lilei, id, domain.ActionResolve, "已处理"); err != nil {
		t.Fatal(err)
	}
	step(t, "resolve", s, id, domain.StatusResolved, 2, 4, "status_changed", `{"from":"in_progress","to":"resolved","note":"已处理"}`)

	_, err = svc.Reassign(ctx, sup, id, hanmei.ID)
	wantCode(t, err, domain.CodeInvalidState)
	step(t, "resolved 改派失败后", s, id, domain.StatusResolved, 2, 4, "status_changed", `{"from":"in_progress","to":"resolved","note":"已处理"}`)

	if _, err := svc.Transition(ctx, lilei, id, domain.ActionReopen, ""); err != nil {
		t.Fatal(err)
	}
	step(t, "reopen", s, id, domain.StatusInProgress, 2, 5, "status_changed", `{"from":"resolved","to":"in_progress","note":""}`)

	_, err = svc.Reassign(ctx, sup, id, lilei.ID)
	wantCode(t, err, domain.CodeSameAssignee)
	_, err = svc.Reassign(ctx, sup, id, sup.ID)
	wantCode(t, err, domain.CodeInvalidAssignee)
	if _, err := svc.Reassign(ctx, sup, id, hanmei.ID); err != nil {
		t.Fatal(err)
	}
	step(t, "reassign", s, id, domain.StatusInProgress, 3, 6, "reassigned", `{"from_assignee_id":2,"to_assignee_id":3}`)

	_, err = svc.Transition(ctx, lilei, id, domain.ActionResolve, "")
	wantCode(t, err, domain.CodeNotAssignee)
	if _, err := svc.Transition(ctx, hanmei, id, domain.ActionResolve, ""); err != nil {
		t.Fatal(err)
	}
	step(t, "resolve 2", s, id, domain.StatusResolved, 3, 7, "status_changed", `{"from":"in_progress","to":"resolved","note":""}`)

	if _, err := svc.Transition(ctx, sup, id, domain.ActionClose, ""); err != nil {
		t.Fatal(err)
	}
	step(t, "close", s, id, domain.StatusClosed, 3, 8, "status_changed", `{"from":"resolved","to":"closed","note":""}`)

	_, err = svc.AddComment(ctx, sup, id, "关闭后评论")
	wantCode(t, err, domain.CodeInvalidState)
	_, err = svc.Transition(ctx, sup, id, domain.ActionReopen, "")
	wantCode(t, err, domain.CodeInvalidTransition)
	_, err = svc.Assign(ctx, sup, id, lilei.ID)
	wantCode(t, err, domain.CodeAlreadyClaimed)
	step(t, "终态失败后", s, id, domain.StatusClosed, 3, 8, "status_changed", `{"from":"resolved","to":"closed","note":""}`)
}

func TestNotFoundAndValidation(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.Claim(ctx, lilei, 9999)
	wantCode(t, err, domain.CodeTicketNotFound)
	_, err = svc.CreateTicket(ctx, sup, domain.CreateTicketInput{Title: "  "})
	wantCode(t, err, domain.CodeValidationError)
	tk := newTicket(t, svc)
	_, err = svc.Assign(ctx, sup, tk.ID, 999)
	wantCode(t, err, domain.CodeInvalidAssignee)
	_, err = svc.Assign(ctx, sup, tk.ID, 0)
	wantCode(t, err, domain.CodeValidationError)
	_, err = svc.Claim(ctx, sup, tk.ID)
	wantCode(t, err, domain.CodeForbidden)
}
