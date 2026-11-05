//go:build integration

package service_test

import (
	"context"
	"testing"
	"time"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/service"
	"ticketdesk/internal/store"
	"ticketdesk/internal/testutil"
)

var ctx = context.Background()

// 与 testutil 插入的固定用户一致（WF§1）。
var (
	sup    = domain.User{ID: 1, Name: "主管 王敏", Role: domain.RoleSupervisor}
	lilei  = domain.User{ID: 2, Name: "客服 李雷", Role: domain.RoleAgent}
	hanmei = domain.User{ID: 3, Name: "客服 韩梅", Role: domain.RoleAgent}
	zhang  = domain.User{ID: 4, Name: "客服 张伟", Role: domain.RoleAgent}
)

func newSvc(t *testing.T) (*service.Service, *store.Store) {
	s := testutil.NewTestStore(t)
	return service.New(s, nil), s
}

func newTicket(t *testing.T, svc *service.Service) *domain.Ticket {
	t.Helper()
	tk, err := svc.CreateTicket(ctx, sup, domain.CreateTicketInput{
		Title: "支付失败", Description: "无法下单", CustomerName: "客户甲",
		CustomerContact: "a@b.c", Priority: domain.PriorityHigh,
	})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func wantCode(t *testing.T, err error, code domain.Code) {
	t.Helper()
	de, ok := err.(*domain.Error)
	if !ok || de.Code != code {
		t.Fatalf("期望 %s，实际 %v", code, err)
	}
}

func events(t *testing.T, q store.Querier, id int64) []store.Event {
	t.Helper()
	ev, err := q.ListEvents(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func countType(evs []store.Event, typ string) int {
	n := 0
	for _, e := range evs {
		if e.Type == typ {
			n++
		}
	}
	return n
}

func fixedClock(at time.Time) func() time.Time { return func() time.Time { return at } }

// fixedNow 截到微秒：PostgreSQL 的 timestamptz 只保留微秒，纳秒会让读回的值对不上。
func fixedNow() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
