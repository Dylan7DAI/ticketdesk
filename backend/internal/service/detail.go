package service

import (
	"context"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// Detail 返回写操作成功后的最新详情（HTTP 层写接口统一返回 TicketDetail，WF§17.2）。
// 调用方已通过各自的 Authorize，这里不再重复判断可见性。
func (s *Service) Detail(ctx context.Context, actor domain.User, t *domain.Ticket) (*TicketDetail, error) {
	return s.buildDetail(ctx, s.db.Q(), actor, t)
}

func (s *Service) buildDetail(ctx context.Context, q store.Querier, actor domain.User, t *domain.Ticket) (*TicketDetail, error) {
	n, err := userNames(ctx, q)
	if err != nil {
		return nil, err
	}
	comments, err := q.ListComments(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	events, err := q.ListEvents(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	var assignee *UserRef
	if t.AssigneeID != nil {
		a := userRef(n, *t.AssigneeID)
		assignee = &a
	}
	d := &TicketDetail{
		TicketSummary:   summarize(*t, assignee, s.now()),
		Description:     t.Description,
		CustomerContact: t.CustomerContact,
		CreatedBy:       userRef(n, t.CreatedBy),
		UpdatedAt:       t.UpdatedAt,
		Comments:        make([]CommentView, len(comments)),
		Events:          make([]EventView, len(events)),
		AllowedActions:  domain.AllowedActions(actor, *t),
	}
	for i, c := range comments {
		d.Comments[i] = CommentView{ID: c.ID, Author: userRef(n, c.AuthorID), Body: c.Body, CreatedAt: c.CreatedAt}
	}
	for i, e := range events {
		d.Events[i] = EventView{
			ID: e.ID, Actor: userRef(n, e.ActorID), Type: e.Type,
			Payload: e.Payload, Summary: Summary(e, n), CreatedAt: e.CreatedAt,
		}
	}
	return d, nil
}
