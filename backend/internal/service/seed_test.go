//go:build integration

package service_test

import (
	"encoding/json"
	"testing"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
	"ticketdesk/internal/testutil"
)

// TestSeedConsistency 用 domain.CanTransition 回放 seed 中每张工单的事件，
// 断言没有非法流转，且回放终点与工单当前的状态和处理人一致（T1.2 遗留的验收）。
func TestSeedConsistency(t *testing.T) {
	s := testutil.NewSeededStore(t)
	tickets, _, err := s.Q().ListTickets(ctx, store.TicketFilter{PageSize: 100})
	if err != nil || len(tickets) != 6 {
		t.Fatalf("seed 应有 6 张工单: %d %v", len(tickets), err)
	}
	for _, tk := range tickets {
		status := domain.StatusOpen
		var assignee *int64
		for _, e := range events(t, s.Q(), tk.ID) {
			var p struct {
				AssigneeID int64         `json:"assignee_id"`
				From       domain.Status `json:"from"`
				To         domain.Status `json:"to"`
			}
			_ = json.Unmarshal(e.Payload, &p)
			switch e.Type {
			case "claimed", "assigned":
				if !domain.CanTransition(status, domain.StatusInProgress) {
					t.Errorf("工单 %d: %s 从 %s 开始是非法流转", tk.ID, e.Type, status)
				}
				status, assignee = domain.StatusInProgress, &p.AssigneeID
			case "status_changed":
				if p.From != status || !domain.CanTransition(p.From, p.To) {
					t.Errorf("工单 %d: 非法流转 %s→%s（回放状态 %s）", tk.ID, p.From, p.To, status)
				}
				status = p.To
			}
		}
		if status != tk.Status || (assignee == nil) != (tk.AssigneeID == nil) ||
			(assignee != nil && *assignee != *tk.AssigneeID) {
			t.Errorf("工单 %d: 回放终点 %s/%v 与库中 %s/%v 不一致", tk.ID, status, assignee, tk.Status, tk.AssigneeID)
		}
	}
}
