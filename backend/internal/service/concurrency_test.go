//go:build integration

package service_test

import (
	"sync"
	"testing"

	"ticketdesk/internal/domain"
)

type result struct {
	who int64
	err error
}

// race 让所有 fn 在同一个闸门后同时起跑。
func race(fns []func() result) []result {
	gate := make(chan struct{})
	out := make([]result, len(fns))
	var wg sync.WaitGroup
	for i, fn := range fns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			out[i] = fn()
		}()
	}
	close(gate)
	wg.Wait()
	return out
}

// I1：20 个 goroutine 用 3 个 agent 轮流 claim 同一工单，恰好 1 个成功。
func TestConcurrentClaim(t *testing.T) {
	svc, s := newSvc(t)
	tk := newTicket(t, svc)
	agents := []domain.User{lilei, hanmei, zhang}

	fns := make([]func() result, 20)
	for i := range fns {
		a := agents[i%len(agents)]
		fns[i] = func() result {
			_, err := svc.Claim(ctx, a, tk.ID)
			return result{a.ID, err}
		}
	}
	var winner int64
	ok, lost := 0, 0
	for _, r := range race(fns) {
		switch {
		case r.err == nil:
			ok++
			winner = r.who
		case r.err.(*domain.Error).Code == domain.CodeAlreadyClaimed:
			lost++
		default:
			t.Fatalf("意外错误: %v", r.err)
		}
	}
	if ok != 1 || lost != 19 {
		t.Fatalf("应 1 成功 19 失败，实际 %d / %d", ok, lost)
	}
	got, _ := s.Q().GetTicket(ctx, tk.ID)
	if got.AssigneeID == nil || *got.AssigneeID != winner || got.Status != domain.StatusInProgress {
		t.Fatalf("库中处理人应为成功者 %d: %+v", winner, got)
	}
	if n := countType(events(t, s.Q(), tk.ID), "claimed"); n != 1 {
		t.Fatalf("claimed 事件应恰好 1 条，实际 %d", n)
	}
}

// I2：agent claim 与 supervisor assign 并发，只有一方成功。
func TestConcurrentClaimVsAssign(t *testing.T) {
	svc, s := newSvc(t)
	tk := newTicket(t, svc)
	res := race([]func() result{
		func() result { _, err := svc.Claim(ctx, lilei, tk.ID); return result{lilei.ID, err} },
		func() result { _, err := svc.Assign(ctx, sup, tk.ID, hanmei.ID); return result{hanmei.ID, err} },
	})
	var winner int64
	ok := 0
	for _, r := range res {
		if r.err == nil {
			ok++
			winner = r.who
		} else {
			wantCode(t, r.err, domain.CodeAlreadyClaimed)
		}
	}
	if ok != 1 {
		t.Fatalf("应恰好一方成功，实际 %d", ok)
	}
	got, _ := s.Q().GetTicket(ctx, tk.ID)
	if *got.AssigneeID != winner {
		t.Fatalf("处理人应为成功方 %d，实际 %d", winner, *got.AssigneeID)
	}
	evs := events(t, s.Q(), tk.ID)
	if countType(evs, "claimed")+countType(evs, "assigned") != 1 {
		t.Fatalf("claimed 与 assigned 事件合计应为 1: %+v", evs)
	}
}
