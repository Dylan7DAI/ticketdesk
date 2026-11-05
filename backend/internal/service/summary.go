package service

import (
	"encoding/json"
	"strings"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

var priorityZH = map[domain.Priority]string{
	domain.PriorityUrgent: "紧急", domain.PriorityHigh: "高",
	domain.PriorityNormal: "普通", domain.PriorityLow: "低",
}

var statusZH = map[domain.Status]string{
	domain.StatusOpen: "待处理", domain.StatusInProgress: "处理中",
	domain.StatusResolved: "已解决", domain.StatusClosed: "已关闭",
}

// shortName 去掉演示数据名称里的角色前缀（"客服 李雷" → "李雷"），摘要读起来更自然（WF§17.3 示例）。
func shortName(full string) string {
	if i := strings.IndexByte(full, ' '); i >= 0 && i+1 < len(full) {
		return full[i+1:]
	}
	return full
}

// nameOf 在批量查出的用户表里取名；用户已不存在时退化为"用户 #id"。
type names map[int64]string

func (n names) of(id int64) string {
	if v, ok := n[id]; ok {
		return shortName(v)
	}
	return "用户 #" + itoa(id)
}

// Summary 按 WF§17.3 模板渲染一条事件的中文说明。payload 解析失败时只返回操作者和类型，不报错。
func Summary(e store.Event, n names) string {
	actor := n.of(e.ActorID)
	var p struct {
		Priority   domain.Priority `json:"priority"`
		AssigneeID int64           `json:"assignee_id"`
		From       json.RawMessage `json:"from"`
		To         json.RawMessage `json:"to"`
		FromID     int64           `json:"from_assignee_id"`
		ToID       int64           `json:"to_assignee_id"`
		Note       string          `json:"note"`
		Excerpt    string          `json:"excerpt"`
	}
	_ = json.Unmarshal(e.Payload, &p)

	switch e.Type {
	case EventCreated:
		return actor + " 创建了工单，优先级 " + priorityZH[p.Priority]
	case EventClaimed:
		return actor + " 领取了工单"
	case EventAssigned:
		return actor + " 将工单分配给 " + n.of(p.AssigneeID)
	case EventReassigned:
		return actor + " 将工单从 " + n.of(p.FromID) + " 改派给 " + n.of(p.ToID)
	case EventStatusChanged:
		s := actor + " 将状态从 " + statusName(p.From) + " 改为 " + statusName(p.To)
		if p.Note != "" {
			s += "：" + p.Note
		}
		return s
	case EventCommented:
		return actor + " 发表评论：" + p.Excerpt
	}
	return actor + " 执行了 " + e.Type
}

func statusName(raw json.RawMessage) string {
	var s domain.Status
	_ = json.Unmarshal(raw, &s)
	if zh, ok := statusZH[s]; ok {
		return zh
	}
	return string(s)
}
