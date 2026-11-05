package service

import (
	"context"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// checkTarget 是 WF§5 第 8 步：目标用户须存在、是 agent，改派时不能等于当前处理人。
func checkTarget(ctx context.Context, q store.Querier, t *domain.Ticket, assigneeID int64) error {
	target, err := q.GetUser(ctx, assigneeID)
	if err != nil {
		return err
	}
	return domain.CheckAssignee(*t, target)
}

func validAssigneeID(id int64) error {
	if id <= 0 {
		return domain.ValidationError(map[string]string{"assignee_id": "处理人 ID 必须是正整数"})
	}
	return nil
}

// Assign 由 supervisor 把未分配的 open 工单分配给 agent，同时进入 in_progress。
func (s *Service) Assign(ctx context.Context, actor domain.User, ticketID, assigneeID int64) (*domain.Ticket, error) {
	if err := validAssigneeID(assigneeID); err != nil {
		return nil, err
	}
	var out *domain.Ticket
	err := s.db.InTx(ctx, func(q store.Querier) error {
		t, err := lockAndAuthorize(ctx, q, actor, ticketID, domain.ActionAssign)
		if err != nil {
			return err
		}
		if err := checkTarget(ctx, q, t, assigneeID); err != nil {
			return err
		}
		if err := q.UpdateTicketState(ctx, ticketID, domain.StatusInProgress, &assigneeID); err != nil {
			return err
		}
		if err := addEvent(ctx, q, ticketID, actor.ID, EventAssigned, assigneePayload{assigneeID}); err != nil {
			return err
		}
		out, err = q.GetTicket(ctx, ticketID)
		return err
	})
	return out, err
}

// Reassign 由 supervisor 把 in_progress 工单改派给另一名 agent，状态不变。
func (s *Service) Reassign(ctx context.Context, actor domain.User, ticketID, assigneeID int64) (*domain.Ticket, error) {
	if err := validAssigneeID(assigneeID); err != nil {
		return nil, err
	}
	var out *domain.Ticket
	err := s.db.InTx(ctx, func(q store.Querier) error {
		t, err := lockAndAuthorize(ctx, q, actor, ticketID, domain.ActionReassign)
		if err != nil {
			return err
		}
		if err := checkTarget(ctx, q, t, assigneeID); err != nil {
			return err
		}
		from := *t.AssigneeID
		if err := q.UpdateTicketState(ctx, ticketID, t.Status, &assigneeID); err != nil {
			return err
		}
		if err := addEvent(ctx, q, ticketID, actor.ID, EventReassigned, reassignedPayload{from, assigneeID}); err != nil {
			return err
		}
		out, err = q.GetTicket(ctx, ticketID)
		return err
	})
	return out, err
}
