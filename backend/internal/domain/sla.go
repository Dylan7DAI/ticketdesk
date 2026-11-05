package domain

import "time"

// slaDurations 是各优先级的 SLA 时长（WF§8）。
var slaDurations = map[Priority]time.Duration{
	PriorityUrgent: 2 * time.Hour,
	PriorityHigh:   8 * time.Hour,
	PriorityNormal: 24 * time.Hour,
	PriorityLow:    72 * time.Hour,
}

// SLADueAt 返回 createdAt 加上该优先级 SLA 时长后的截止时间。
// 优先级非法时按 normal 处理；输入已由 Normalize 校验，这里只是防御。
func SLADueAt(createdAt time.Time, p Priority) time.Time {
	d, ok := slaDurations[p]
	if !ok {
		d = slaDurations[PriorityNormal]
	}
	return createdAt.Add(d)
}

// IsOverdue 判断工单在 now 时刻是否逾期：状态为 open 或 in_progress，且 now 严格晚于截止时间。
// resolved、closed 永不逾期；reopen 后按原截止时间继续判断（WF§8）。
func IsOverdue(t Ticket, now time.Time) bool {
	if t.Status != StatusOpen && t.Status != StatusInProgress {
		return false
	}
	return now.After(t.SLADueAt)
}
