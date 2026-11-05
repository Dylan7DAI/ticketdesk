package domain

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)

// U3：4 个优先级的 SLA 时长。
func TestSLADueAt(t *testing.T) {
	cases := map[Priority]time.Duration{
		PriorityUrgent: 2 * time.Hour,
		PriorityHigh:   8 * time.Hour,
		PriorityNormal: 24 * time.Hour,
		PriorityLow:    72 * time.Hour,
	}
	for p, want := range cases {
		if got := SLADueAt(t0, p).Sub(t0); got != want {
			t.Errorf("%s: 期望 %v，实际 %v", p, want, got)
		}
	}
}

// U4：截止前、恰好截止、截止后；按状态区分是否逾期。
func TestIsOverdue(t *testing.T) {
	due := SLADueAt(t0, PriorityHigh)
	cases := []struct {
		name   string
		status Status
		now    time.Time
		want   bool
	}{
		{"open/截止前1秒", StatusOpen, due.Add(-time.Second), false},
		{"open/恰好截止", StatusOpen, due, false},
		{"open/截止后1秒", StatusOpen, due.Add(time.Second), true},
		{"in_progress/截止前1秒", StatusInProgress, due.Add(-time.Second), false},
		{"in_progress/恰好截止", StatusInProgress, due, false},
		{"in_progress/截止后1秒", StatusInProgress, due.Add(time.Second), true},
		{"resolved/截止后1秒", StatusResolved, due.Add(time.Second), false},
		{"closed/截止后1秒", StatusClosed, due.Add(time.Second), false},
		{"resolved/远超截止", StatusResolved, due.Add(1000 * time.Hour), false},
		{"closed/远超截止", StatusClosed, due.Add(1000 * time.Hour), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tk := Ticket{Status: c.status, CreatedAt: t0, SLADueAt: due}
			if got := IsOverdue(tk, c.now); got != c.want {
				t.Fatalf("期望 %v，实际 %v", c.want, got)
			}
		})
	}
}

// seed 口径（WF§10）：恰好 2 张逾期，另外 4 张不逾期。
func TestIsOverdue_SeedScenario(t *testing.T) {
	now := t0
	mk := func(ago time.Duration, p Priority, s Status) Ticket {
		c := now.Add(-ago)
		return Ticket{Status: s, Priority: p, CreatedAt: c, SLADueAt: SLADueAt(c, p)}
	}
	seed := []struct {
		name string
		t    Ticket
		want bool
	}{
		{"支付失败无法下单", mk(3*time.Hour, PriorityUrgent, StatusOpen), true},
		{"发票抬头修改", mk(1*time.Hour, PriorityNormal, StatusOpen), false},
		{"物流长时间未更新", mk(10*time.Hour, PriorityHigh, StatusInProgress), true},
		{"账号无法登录", mk(2*time.Hour, PriorityHigh, StatusInProgress), false},
		{"退款进度查询", mk(30*time.Hour, PriorityNormal, StatusResolved), false},
		{"修改收货地址", mk(120*time.Hour, PriorityLow, StatusClosed), false},
	}
	overdue := 0
	for _, s := range seed {
		got := IsOverdue(s.t, now)
		if got != s.want {
			t.Errorf("%s: 期望 %v，实际 %v", s.name, s.want, got)
		}
		if got {
			overdue++
		}
	}
	if overdue != 2 {
		t.Fatalf("seed 应恰好 2 张逾期，实际 %d", overdue)
	}
}

// 非法优先级按 normal 处理（防御分支，正常输入已被 Normalize 拦截）。
func TestSLADueAt_UnknownPriorityFallsBackToNormal(t *testing.T) {
	if got := SLADueAt(t0, Priority("critical")).Sub(t0); got != 24*time.Hour {
		t.Fatalf("期望 24h，实际 %v", got)
	}
}
