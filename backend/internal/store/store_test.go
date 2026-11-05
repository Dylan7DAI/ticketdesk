//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
	"ticketdesk/internal/testutil"
)

var ctx = context.Background()

// mk 插入一张工单；assignee 非 nil 时状态为 in_progress，否则 open。
func mk(t *testing.T, q store.Querier, title string, assignee *int64, age time.Duration) *domain.Ticket {
	t.Helper()
	created := time.Now().Add(-age)
	tk := &domain.Ticket{
		Title: title, Description: "描述", CustomerName: "客户甲", CustomerContact: "a@b.c",
		Priority: domain.PriorityNormal, Status: domain.StatusOpen, AssigneeID: assignee,
		CreatedBy: 1, SLADueAt: created.Add(24 * time.Hour), CreatedAt: created, UpdatedAt: created,
	}
	if assignee != nil {
		tk.Status = domain.StatusInProgress
	}
	if err := q.InsertTicket(ctx, tk); err != nil {
		t.Fatal(err)
	}
	return tk
}

func id(v int64) *int64 { return &v }

func TestLockTicketBlocks(t *testing.T) {
	s := testutil.NewTestStore(t)
	tk := mk(t, s.Q(), "支付失败", nil, time.Hour)

	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.InTx(ctx, func(q store.Querier) error {
			if _, err := q.LockTicket(ctx, tk.ID); err != nil {
				return err
			}
			close(locked)
			<-release
			return q.UpdateTicketState(ctx, tk.ID, domain.StatusInProgress, id(2))
		})
	}()
	<-locked

	got := make(chan *domain.Ticket, 1)
	go func() {
		_ = s.InTx(ctx, func(q store.Querier) error {
			x, err := q.LockTicket(ctx, tk.ID)
			got <- x
			return err
		})
	}()

	select {
	case <-got:
		t.Fatal("事务 A 未提交时，事务 B 的 LockTicket 不应返回")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	select {
	case x := <-got:
		if x.Status != domain.StatusInProgress || x.AssigneeID == nil || *x.AssigneeID != 2 {
			t.Fatalf("B 应读到 A 提交后的最新数据，实际 %+v", x)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("A 提交后 B 仍未返回")
	}
}

func TestInTxRollbackOnError(t *testing.T) {
	s := testutil.NewTestStore(t)
	boom := errors.New("boom")
	var tid int64
	err := s.InTx(ctx, func(q store.Querier) error {
		tid = mk(t, q, "会被回滚", nil, time.Minute).ID
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("应原样返回 fn 的错误，实际 %v", err)
	}
	if got, err := s.Q().GetTicket(ctx, tid); err != nil || got != nil {
		t.Fatalf("回滚后工单不应存在: %+v, %v", got, err)
	}
}

func TestInTxRollbackOnPanic(t *testing.T) {
	s := testutil.NewTestStore(t)
	var tid int64
	func() {
		defer func() { _ = recover() }()
		_ = s.InTx(ctx, func(q store.Querier) error {
			tid = mk(t, q, "panic 回滚", nil, time.Minute).ID
			panic("x")
		})
	}()
	if got, _ := s.Q().GetTicket(ctx, tid); got != nil {
		t.Fatal("panic 后事务应回滚")
	}
}
