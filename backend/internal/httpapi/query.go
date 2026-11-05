package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ValidationError(map[string]string{"id": "工单 ID 必须是正整数"})
	}
	return id, nil
}

// splitEnum 解析逗号分隔的枚举，逐个校验；空值返回 nil。
func splitEnum[T ~string](details map[string]string, field, raw string, parse func(string) (T, bool)) []T {
	if raw == "" {
		return nil
	}
	var out []T
	for _, part := range strings.Split(raw, ",") {
		v, ok := parse(part)
		if !ok {
			details[field] = "包含无效的取值：" + part
			return nil
		}
		out = append(out, v)
	}
	return out
}

func intParam(details map[string]string, qs url.Values, field string, def, min, max int) int {
	raw := qs.Get(field)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		details[field] = field + " 必须是 " + strconv.Itoa(min) + " 到 " + strconv.Itoa(max) + " 之间的整数"
		return def
	}
	return n
}

// parseFilter 解析列表参数（WF§11）。非法值返回 400 VALIDATION_ERROR，details 列出全部出错参数。
func parseFilter(qs url.Values) (store.TicketFilter, error) {
	details := map[string]string{}
	f := store.TicketFilter{Q: qs.Get("q")}
	f.Statuses = splitEnum(details, "status", qs.Get("status"), domain.ParseStatus)
	f.Priorities = splitEnum(details, "priority", qs.Get("priority"), domain.ParsePriority)
	f.Page = intParam(details, qs, "page", 1, 1, 1_000_000)
	f.PageSize = intParam(details, qs, "page_size", 20, 1, 100)

	switch raw := qs.Get("assignee_id"); {
	case raw == "":
	case raw == "none":
		f.Assignee = &store.AssigneeFilter{Unassigned: true}
	default:
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			details["assignee_id"] = "assignee_id 必须是正整数或 none"
		} else {
			f.Assignee = &store.AssigneeFilter{UserID: id}
		}
	}
	if len(details) > 0 {
		return store.TicketFilter{}, domain.ValidationError(details)
	}
	return f, nil
}
