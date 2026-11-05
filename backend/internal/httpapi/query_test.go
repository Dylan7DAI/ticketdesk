package httpapi

import (
	"net/url"
	"testing"

	"ticketdesk/internal/domain"
)

func TestParseFilterValid(t *testing.T) {
	qs, _ := url.ParseQuery("status=open,in_progress&priority=high&assignee_id=none&q=支付&page=2&page_size=50")
	f, err := parseFilter(qs)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Statuses) != 2 || f.Statuses[1] != domain.StatusInProgress || len(f.Priorities) != 1 ||
		f.Assignee == nil || !f.Assignee.Unassigned || f.Q != "支付" || f.Page != 2 || f.PageSize != 50 {
		t.Fatalf("解析结果不对: %+v", f)
	}
	f, _ = parseFilter(url.Values{"assignee_id": {"3"}})
	if f.Assignee == nil || f.Assignee.UserID != 3 || f.Page != 1 || f.PageSize != 20 {
		t.Fatalf("assignee_id=3 与默认分页不对: %+v", f)
	}
}

func TestParseFilterInvalid(t *testing.T) {
	cases := map[string]string{
		"status=open,bogus": "status", "priority=critical": "priority",
		"assignee_id=abc": "assignee_id", "assignee_id=0": "assignee_id", "assignee_id=-1": "assignee_id",
		"page=0": "page", "page=x": "page", "page_size=0": "page_size", "page_size=101": "page_size",
		"status=open,": "status", "status=%20open": "status",
	}
	for raw, field := range cases {
		qs, _ := url.ParseQuery(raw)
		_, err := parseFilter(qs)
		de, ok := err.(*domain.Error)
		if !ok || de.Code != domain.CodeValidationError || de.Details[field] == "" {
			t.Errorf("%s: 应为 VALIDATION_ERROR 且 details[%s] 非空，实际 %v", raw, field, err)
		}
	}
}
