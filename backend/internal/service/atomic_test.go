//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/service"
	"ticketdesk/internal/store"
)

var errInjected = errors.New("injected: 写事件失败")

// failingQ 把 InsertEvent 替换成必定失败，其余方法透传给真实 Querier。
type failingQ struct{ store.Querier }

func (failingQ) InsertEvent(context.Context, *store.Event) error { return errInjected }

// failingDB 让 InTx 内拿到的 Querier 都是 failingQ，事务本身仍是真实事务。
type failingDB struct{ *store.Store }

func (f failingDB) InTx(ctx context.Context, fn func(q store.Querier) error) error {
	return f.Store.InTx(ctx, func(q store.Querier) error { return fn(failingQ{q}) })
}

// I7：业务变更与历史必须同事务。事件写失败时，状态、处理人、评论都不能留下痕迹。
func TestAtomicityWhenEventFails(t *testing.T) {
	svc, s := newSvc(t)
	tk := newTicket(t, svc)
	bad := service.New(failingDB{s}, nil)

	if _, err := bad.Claim(ctx, lilei, tk.ID); !errors.Is(err, errInjected) {
		t.Fatalf("Claim 应返回注入的错误，实际 %v", err)
	}
	got, _ := s.Q().GetTicket(ctx, tk.ID)
	if got.Status != domain.StatusOpen || got.AssigneeID != nil {
		t.Fatalf("Claim 失败后工单应保持 open 未分配: %+v", got)
	}

	if _, err := svc.Claim(ctx, lilei, tk.ID); err != nil {
		t.Fatal(err)
	}
	before := len(events(t, s.Q(), tk.ID))

	if _, err := bad.AddComment(ctx, lilei, tk.ID, "不该留下"); !errors.Is(err, errInjected) {
		t.Fatalf("AddComment 应返回注入的错误，实际 %v", err)
	}
	if cs, _ := s.Q().ListComments(ctx, tk.ID); len(cs) != 0 {
		t.Fatalf("评论应随事务回滚，实际 %d 条", len(cs))
	}
	if _, err := bad.Transition(ctx, lilei, tk.ID, domain.ActionResolve, ""); !errors.Is(err, errInjected) {
		t.Fatalf("Transition 应返回注入的错误，实际 %v", err)
	}
	got, _ = s.Q().GetTicket(ctx, tk.ID)
	if got.Status != domain.StatusInProgress || len(events(t, s.Q(), tk.ID)) != before {
		t.Fatalf("Transition 失败后状态与事件数应不变: %+v", got)
	}
}

func TestCreateTicketAtomic(t *testing.T) {
	_, s := newSvc(t)
	bad := service.New(failingDB{s}, nil)
	_, err := bad.CreateTicket(ctx, sup, domain.CreateTicketInput{
		Title: "x", Description: "y", CustomerName: "z", CustomerContact: "w", Priority: domain.PriorityLow,
	})
	if !errors.Is(err, errInjected) {
		t.Fatalf("应返回注入的错误: %v", err)
	}
	if _, total, _ := s.Q().ListTickets(ctx, store.TicketFilter{}); total != 0 {
		t.Fatalf("创建失败后不应留下工单，实际 %d", total)
	}
}

// 创建工单用注入时钟：created_at、updated_at 相同，sla_due_at - created_at 恰为 SLA 时长。
func TestCreateUsesInjectedClock(t *testing.T) {
	_, s := newSvc(t)
	now := fixedNow()
	svc := service.New(s, fixedClock(now))
	tk := newTicket(t, svc)
	if !tk.CreatedAt.Equal(now) || !tk.UpdatedAt.Equal(now) {
		t.Fatalf("created/updated 应等于注入时钟: %v %v", tk.CreatedAt, tk.UpdatedAt)
	}
	if d := tk.SLADueAt.Sub(tk.CreatedAt); d.Hours() != 8 {
		t.Fatalf("high 的 SLA 应为 8h，实际 %v", d)
	}
}
