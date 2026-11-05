//go:build integration

package store_test

import (
	"testing"
	"time"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
	"ticketdesk/internal/testutil"
)

func titles(rows []store.TicketRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Title
	}
	return out
}

// 造数据：A、B 未分配；C 给李雷(2)；D 给韩梅(3)。创建时间 A 最新。
func seedList(t *testing.T) *store.Store {
	s := testutil.NewTestStore(t)
	q := s.Q()
	mk(t, q, "A 未分配", nil, 1*time.Hour)
	mk(t, q, "B 折扣 50% 优惠", nil, 2*time.Hour)
	mk(t, q, "C 李雷的", id(2), 3*time.Hour)
	mk(t, q, "D 韩梅的", id(3), 4*time.Hour)
	return s
}

func TestListVisibleTo(t *testing.T) {
	s := seedList(t)
	items, total, err := s.Q().ListTickets(ctx, store.TicketFilter{VisibleTo: id(2)})
	if err != nil {
		t.Fatal(err)
	}
	got := titles(items)
	want := []string{"A 未分配", "B 折扣 50% 优惠", "C 李雷的"}
	if total != 3 || len(got) != 3 || got[0] != want[0] || got[2] != want[2] {
		t.Fatalf("李雷应看到 3 张（含全部未分配），实际 total=%d %v", total, got)
	}
}

func TestListVisibleToWithOtherAssignee(t *testing.T) {
	s := seedList(t)
	f := store.TicketFilter{VisibleTo: id(2), Assignee: &store.AssigneeFilter{UserID: 3}}
	items, total, err := s.Q().ListTickets(ctx, f)
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("李雷筛选韩梅的工单应为空: total=%d err=%v", total, err)
	}
}

func TestListLikeEscape(t *testing.T) {
	s := seedList(t)
	for q, want := range map[string]int{"50%": 1, "%": 1, "_": 0, `\`: 0, "未分配": 1, "客户甲": 4, "李雷": 1} {
		_, total, err := s.Q().ListTickets(ctx, store.TicketFilter{Q: q})
		if err != nil {
			t.Fatal(err)
		}
		if total != want {
			t.Errorf("q=%q 应匹配 %d 张，实际 %d", q, want, total)
		}
	}
}

func TestListUnassignedAndStatuses(t *testing.T) {
	s := seedList(t)
	_, total, _ := s.Q().ListTickets(ctx, store.TicketFilter{Assignee: &store.AssigneeFilter{Unassigned: true}})
	if total != 2 {
		t.Fatalf("未分配应为 2，实际 %d", total)
	}
	f := store.TicketFilter{Statuses: []domain.Status{domain.StatusOpen, domain.StatusClosed}}
	if _, total, _ = s.Q().ListTickets(ctx, f); total != 2 {
		t.Fatalf("open+closed 应为 2，实际 %d", total)
	}
	f = store.TicketFilter{Priorities: []domain.Priority{domain.PriorityUrgent}}
	if _, total, _ = s.Q().ListTickets(ctx, f); total != 0 {
		t.Fatalf("urgent 应为 0，实际 %d", total)
	}
}

func TestListPagingStable(t *testing.T) {
	s := seedList(t)
	seen := map[string]bool{}
	for page := 1; page <= 2; page++ {
		items, total, err := s.Q().ListTickets(ctx, store.TicketFilter{Page: page, PageSize: 2})
		if err != nil || total != 4 || len(items) != 2 {
			t.Fatalf("page %d: total=%d len=%d err=%v", page, total, len(items), err)
		}
		for _, title := range titles(items) {
			if seen[title] {
				t.Fatalf("翻页出现重复: %s", title)
			}
			seen[title] = true
		}
	}
	if len(seen) != 4 {
		t.Fatalf("翻页应不重复不遗漏，实际 %v", seen)
	}
	if items, total, _ := s.Q().ListTickets(ctx, store.TicketFilter{Page: 3, PageSize: 2}); len(items) != 0 || total != 4 {
		t.Fatal("越界页应返回空 items 且 total 不变")
	}
}

func TestListAssigneeName(t *testing.T) {
	s := seedList(t)
	items, _, _ := s.Q().ListTickets(ctx, store.TicketFilter{Assignee: &store.AssigneeFilter{UserID: 2}})
	if len(items) != 1 || items[0].AssigneeName == nil || *items[0].AssigneeName != "客服 李雷" {
		t.Fatalf("应带出处理人姓名: %+v", items)
	}
}
