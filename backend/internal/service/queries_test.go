//go:build integration

package service_test

import (
	"testing"
	"time"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/service"
	"ticketdesk/internal/store"
	"ticketdesk/internal/testutil"
)

func seeded(t *testing.T) *service.Service {
	return service.New(testutil.NewSeededStore(t), nil)
}

func listTitles(p *service.TicketPage) map[string]bool {
	m := map[string]bool{}
	for _, it := range p.Items {
		m[it.Title] = true
	}
	return m
}

// I9：李雷看不到韩梅的工单，能看到全部 open 未分配；传 assignee_id=3 得到空列表。
func TestListVisibilityForAgent(t *testing.T) {
	svc := seeded(t)
	p, err := svc.ListTickets(ctx, lilei, store.TicketFilter{PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	got := listTitles(p)
	for _, want := range []string{"支付失败无法下单", "发票抬头修改", "物流长时间未更新", "退款进度查询"} {
		if !got[want] {
			t.Errorf("李雷应看到 %q", want)
		}
	}
	if got["账号无法登录"] || got["修改收货地址"] || p.Total != 4 {
		t.Errorf("李雷不应看到韩梅、张伟的工单: total=%d %v", p.Total, got)
	}

	f := store.TicketFilter{Assignee: &store.AssigneeFilter{UserID: hanmei.ID}}
	if p, _ = svc.ListTickets(ctx, lilei, f); p.Total != 0 || len(p.Items) != 0 {
		t.Errorf("李雷传 assignee_id=3 应为空，实际 %d", p.Total)
	}
}

// 调用方传入的 VisibleTo 不能被利用：agent 无法伪造，supervisor 会被清除。
func TestListVisibleToCannotBeSpoofed(t *testing.T) {
	svc := seeded(t)
	other := hanmei.ID
	p, _ := svc.ListTickets(ctx, lilei, store.TicketFilter{VisibleTo: &other, PageSize: 100})
	if listTitles(p)["账号无法登录"] {
		t.Fatal("agent 传入他人的 VisibleTo 不应生效")
	}
	p, _ = svc.ListTickets(ctx, sup, store.TicketFilter{VisibleTo: &other, PageSize: 100})
	if p.Total != 6 {
		t.Fatalf("supervisor 应看到全部 6 张，实际 %d", p.Total)
	}
}

func TestListFiltersForSupervisor(t *testing.T) {
	svc := seeded(t)
	cases := []struct {
		name string
		f    store.TicketFilter
		want int
	}{
		{"全部", store.TicketFilter{}, 6},
		{"status 多选", store.TicketFilter{Statuses: []domain.Status{domain.StatusOpen, domain.StatusInProgress}}, 4},
		{"未分配", store.TicketFilter{Assignee: &store.AssigneeFilter{Unassigned: true}}, 2},
		{"按处理人", store.TicketFilter{Assignee: &store.AssigneeFilter{UserID: lilei.ID}}, 2},
		{"优先级", store.TicketFilter{Priorities: []domain.Priority{domain.PriorityHigh}}, 2},
		{"搜标题", store.TicketFilter{Q: "支付"}, 1},
		{"搜客户名", store.TicketFilter{Q: "ZZZZ"}, 0},
	}
	for _, c := range cases {
		p, err := svc.ListTickets(ctx, sup, c.f)
		if err != nil || p.Total != c.want {
			t.Errorf("%s: 期望 %d，实际 %v %v", c.name, c.want, p, err)
		}
	}
}

func TestListDefaultPaging(t *testing.T) {
	p, err := seeded(t).ListTickets(ctx, sup, store.TicketFilter{})
	if err != nil || p.Page != 1 || p.PageSize != 20 || len(p.Items) != 6 {
		t.Fatalf("默认分页应为 1/20: %+v %v", p, err)
	}
	if p.Items[0].CreatedAt.Before(p.Items[5].CreatedAt) {
		t.Fatal("默认应按创建时间倒序")
	}
}

// 按 WF§8 口径 seed 中恰好 2 张逾期，且 resolved/closed 不逾期。
func TestSeedOverdue(t *testing.T) {
	p, _ := seeded(t).ListTickets(ctx, sup, store.TicketFilter{PageSize: 100})
	overdue := map[string]bool{}
	for _, it := range p.Items {
		if it.IsOverdue {
			overdue[it.Title] = true
		}
	}
	if len(overdue) != 2 || !overdue["支付失败无法下单"] || !overdue["物流长时间未更新"] {
		t.Fatalf("逾期应恰为支付失败、物流未更新: %v", overdue)
	}
}

// 注入时钟让刚创建的工单刚过截止时间，is_overdue 为 true；resolve 后为 false。
func TestOverdueWithInjectedClock(t *testing.T) {
	_, s := newSvc(t)
	created := fixedNow()
	tk := newTicket(t, service.New(s, fixedClock(created)))
	late := service.New(s, fixedClock(tk.SLADueAt.Add(time.Second)))

	d, err := late.GetTicketDetail(ctx, sup, tk.ID)
	if err != nil || !d.IsOverdue {
		t.Fatalf("过了截止时间应逾期: %+v %v", d, err)
	}
	onTime := service.New(s, fixedClock(tk.SLADueAt))
	if d, _ = onTime.GetTicketDetail(ctx, sup, tk.ID); d.IsOverdue {
		t.Fatal("恰好等于截止时间不算逾期")
	}
	if _, err := late.Claim(ctx, lilei, tk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := late.Transition(ctx, lilei, tk.ID, domain.ActionResolve, ""); err != nil {
		t.Fatal(err)
	}
	if d, _ = late.GetTicketDetail(ctx, sup, tk.ID); d.IsOverdue {
		t.Fatal("resolve 后不应再逾期")
	}
}
