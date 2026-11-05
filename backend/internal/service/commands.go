package service

import (
	"context"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// lockAndAuthorize 是 WF§5 第 4～7 步：锁行 → 404 → 角色/归属/状态判定。
// 锁住之后读到的就是最终判断依据，所以 404、403、409 的先后顺序是确定的。
func lockAndAuthorize(ctx context.Context, q store.Querier, actor domain.User, id int64, a domain.Action) (*domain.Ticket, error) {
	t, err := q.LockTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domain.ErrTicketNotFound()
	}
	if err := domain.Authorize(actor, *t, a); err != nil {
		return nil, err
	}
	return t, nil
}

// CreateTicket 创建工单。取一次 now 同时作为 created_at、updated_at 和 SLA 起点（WF§17.2）。
func (s *Service) CreateTicket(ctx context.Context, actor domain.User, in domain.CreateTicketInput) (*domain.Ticket, error) {
	in, err := in.Normalize()
	if err != nil {
		return nil, err
	}
	now := s.now()
	t := &domain.Ticket{
		Title: in.Title, Description: in.Description,
		CustomerName: in.CustomerName, CustomerContact: in.CustomerContact,
		Priority: in.Priority, Status: domain.StatusOpen, CreatedBy: actor.ID,
		SLADueAt: domain.SLADueAt(now, in.Priority), CreatedAt: now, UpdatedAt: now,
	}
	err = s.db.InTx(ctx, func(q store.Querier) error {
		if err := q.InsertTicket(ctx, t); err != nil {
			return err
		}
		return addEvent(ctx, q, t.ID, actor.ID, EventCreated, createdPayload{in.Priority})
	})
	if err != nil {
		return nil, err
	}
	// 从库读回：Go 的纳秒时间写入 timestamptz 后只保留微秒，返回值应与库中一致。
	return s.db.Q().GetTicket(ctx, t.ID)
}

// Claim 由 agent 领取未分配的 open 工单。
func (s *Service) Claim(ctx context.Context, actor domain.User, ticketID int64) (*domain.Ticket, error) {
	var out *domain.Ticket
	err := s.db.InTx(ctx, func(q store.Querier) error {
		if _, err := lockAndAuthorize(ctx, q, actor, ticketID, domain.ActionClaim); err != nil {
			return err
		}
		if err := q.UpdateTicketState(ctx, ticketID, domain.StatusInProgress, &actor.ID); err != nil {
			return err
		}
		if err := addEvent(ctx, q, ticketID, actor.ID, EventClaimed, assigneePayload{actor.ID}); err != nil {
			return err
		}
		var err error
		out, err = q.GetTicket(ctx, ticketID)
		return err
	})
	return out, err
}
