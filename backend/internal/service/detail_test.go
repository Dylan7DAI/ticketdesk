//go:build integration

package service_test

import (
	"reflect"
	"testing"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/service"
	"ticketdesk/internal/store"
)

func ticketID(t *testing.T, svc *service.Service, title string) int64 {
	t.Helper()
	p, err := svc.ListTickets(ctx, sup, store.TicketFilter{Q: title})
	if err != nil || len(p.Items) != 1 {
		t.Fatalf("找不到工单 %q: %v", title, err)
	}
	return p.Items[0].ID
}

func TestDetailAccess(t *testing.T) {
	svc := seeded(t)
	other := ticketID(t, svc, "账号无法登录") // 韩梅
	// 详情对所有客服只读可见（WF§4）；看到不等于能操作，操作权限见 allowed_actions。
	d, err := svc.GetTicketDetail(ctx, lilei, other)
	if err != nil {
		t.Fatalf("客服应能只读查看他人工单: %v", err)
	}
	if len(d.AllowedActions) != 0 {
		t.Fatalf("非处理人的客服不应有任何动作: %v", d.AllowedActions)
	}
	_, err = svc.GetTicketDetail(ctx, lilei, 9999)
	wantCode(t, err, domain.CodeTicketNotFound)
	if _, err = svc.GetTicketDetail(ctx, hanmei, other); err != nil {
		t.Fatalf("处理人应能查看: %v", err)
	}
	if _, err = svc.GetTicketDetail(ctx, sup, other); err != nil {
		t.Fatalf("主管应能查看: %v", err)
	}
	open := ticketID(t, svc, "发票抬头修改")
	if _, err = svc.GetTicketDetail(ctx, zhang, open); err != nil {
		t.Fatalf("open 未分配对任意客服可见: %v", err)
	}
}

// 详情中的 allowed_actions 与 domain.AllowedActions 一致，并对应 T5.4 的按钮表。
func TestDetailAllowedActions(t *testing.T) {
	svc := seeded(t)
	// 无动作时必须是非 nil 的空切片，序列化为 JSON 才是 [] 而不是 null。
	a := func(x ...domain.Action) []domain.Action { return append([]domain.Action{}, x...) }
	cases := []struct {
		actor domain.User
		title string
		want  []domain.Action
	}{
		{lilei, "发票抬头修改", a(domain.ActionClaim)},
		{sup, "发票抬头修改", a(domain.ActionAssign, domain.ActionComment)},
		{sup, "账号无法登录", a(domain.ActionReassign, domain.ActionResolve, domain.ActionComment)},
		{hanmei, "账号无法登录", a(domain.ActionResolve, domain.ActionComment)},
		{lilei, "退款进度查询", a(domain.ActionReopen, domain.ActionClose, domain.ActionComment)},
		{sup, "修改收货地址", a()},
	}
	for _, c := range cases {
		d, err := svc.GetTicketDetail(ctx, c.actor, ticketID(t, svc, c.title))
		if err != nil {
			t.Fatalf("%s/%s: %v", c.actor.Name, c.title, err)
		}
		if !reflect.DeepEqual(d.AllowedActions, c.want) {
			t.Errorf("%s 看 %s: 期望 %v，实际 %v", c.actor.Name, c.title, c.want, d.AllowedActions)
		}
	}
}

func TestDetailEventsAndSummary(t *testing.T) {
	svc := seeded(t)
	d, err := svc.GetTicketDetail(ctx, sup, ticketID(t, svc, "修改收货地址"))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Events) != 4 || d.Events[0].Type != "created" || d.Events[3].Type != "status_changed" {
		t.Fatalf("closed 工单应有 4 条事件且按时间升序: %+v", d.Events)
	}
	for i := 1; i < len(d.Events); i++ {
		if d.Events[i].CreatedAt.Before(d.Events[i-1].CreatedAt) {
			t.Fatal("事件应按时间升序")
		}
	}
	if got := d.Events[1].Summary; got != "张伟 领取了工单" {
		t.Errorf("领取摘要: %q", got)
	}
	if got := d.Events[3].Summary; got != "张伟 将状态从 已解决 改为 已关闭" {
		t.Errorf("无备注的关闭摘要: %q", got)
	}
	if got := d.Events[2].Summary; got != "张伟 将状态从 处理中 改为 已解决：已按客户要求修改地址" {
		t.Errorf("带备注的摘要: %q", got)
	}
	if d.Assignee == nil || d.Assignee.Name != "客服 张伟" || d.CreatedBy.ID == 0 {
		t.Errorf("应带出处理人与创建人: %+v", d.TicketSummary)
	}
}

func TestDetailWithCommentsAfterWrites(t *testing.T) {
	svc, _ := newSvc(t)
	tk := newTicket(t, svc)
	_, _ = svc.Assign(ctx, sup, tk.ID, lilei.ID)
	_, _ = svc.Reassign(ctx, sup, tk.ID, hanmei.ID)
	if _, err := svc.AddComment(ctx, hanmei, tk.ID, "跟进中"); err != nil {
		t.Fatal(err)
	}
	d, _ := svc.GetTicketDetail(ctx, sup, tk.ID)
	want := []string{
		"王敏 创建了工单，优先级 高",
		"王敏 将工单分配给 李雷",
		"王敏 将工单从 李雷 改派给 韩梅",
		"韩梅 发表评论：跟进中",
	}
	if len(d.Events) != 4 || len(d.Comments) != 1 || d.Comments[0].Author.Name != "客服 韩梅" {
		t.Fatalf("事件/评论数不对: %+v", d)
	}
	for i, w := range want {
		if d.Events[i].Summary != w {
			t.Errorf("事件 %d 摘要: 期望 %q，实际 %q", i, w, d.Events[i].Summary)
		}
	}
}
