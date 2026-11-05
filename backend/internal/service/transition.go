package service

import (
	"context"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// Transition 执行 resolve / reopen / close。目标状态来自 domain.TargetStatus，
// 起始状态的检查在 Authorize 中按动作固定起始状态判断（WF§17.1），不在这里重复。
func (s *Service) Transition(ctx context.Context, actor domain.User, ticketID int64, a domain.Action, note string) (*domain.Ticket, error) {
	to, ok := domain.TargetStatus(a)
	if !ok {
		return nil, domain.ValidationError(map[string]string{"action": "不支持的状态流转动作"})
	}
	note, err := domain.NormalizeNote(note)
	if err != nil {
		return nil, err
	}
	var out *domain.Ticket
	err = s.db.InTx(ctx, func(q store.Querier) error {
		t, err := lockAndAuthorize(ctx, q, actor, ticketID, a)
		if err != nil {
			return err
		}
		if err := q.UpdateTicketState(ctx, ticketID, to, t.AssigneeID); err != nil {
			return err
		}
		ev := statusPayload{From: t.Status, To: to, Note: note}
		if err := addEvent(ctx, q, ticketID, actor.ID, EventStatusChanged, ev); err != nil {
			return err
		}
		out, err = q.GetTicket(ctx, ticketID)
		return err
	})
	return out, err
}

// AddComment 添加评论。评论也锁工单行，防止刚被改派走的客服同时在原工单上评论成功（WF§7）。
func (s *Service) AddComment(ctx context.Context, actor domain.User, ticketID int64, body string) (*store.Comment, error) {
	body, err := domain.NormalizeComment(body)
	if err != nil {
		return nil, err
	}
	c := &store.Comment{TicketID: ticketID, AuthorID: actor.ID, Body: body}
	err = s.db.InTx(ctx, func(q store.Querier) error {
		if _, err := lockAndAuthorize(ctx, q, actor, ticketID, domain.ActionComment); err != nil {
			return err
		}
		if err := q.InsertComment(ctx, c); err != nil {
			return err
		}
		ev := commentedPayload{CommentID: c.ID, Excerpt: excerpt(body)}
		return addEvent(ctx, q, ticketID, actor.ID, EventCommented, ev)
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}
