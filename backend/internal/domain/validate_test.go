package domain

import (
	"errors"
	"strings"
	"testing"
)

func validInput() CreateTicketInput {
	return CreateTicketInput{
		Title:           "支付失败无法下单",
		Description:     "用户下单时提示支付失败",
		CustomerName:    "张三",
		CustomerContact: "13800000000",
		Priority:        PriorityNormal,
	}
}

func asValidation(t *testing.T, err error) *Error {
	t.Helper()
	var de *Error
	if !errors.As(err, &de) {
		t.Fatalf("期望 *Error，实际 %T: %v", err, err)
	}
	if de.Code != CodeValidationError {
		t.Fatalf("期望 VALIDATION_ERROR，实际 %s", de.Code)
	}
	return de
}

func TestNormalize_Valid(t *testing.T) {
	in := validInput()
	in.Title = "  标题  "
	in.Description = "\t描述\n"
	in.CustomerName = " 客户 "
	in.CustomerContact = " 联系方式 "
	out, err := in.Normalize()
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if out.Title != "标题" || out.Description != "描述" ||
		out.CustomerName != "客户" || out.CustomerContact != "联系方式" {
		t.Fatalf("首尾空白未去除: %+v", out)
	}
}

func TestNormalize_BlankTitle(t *testing.T) {
	in := validInput()
	in.Title = "   "
	_, err := in.Normalize()
	de := asValidation(t, err)
	if de.Details["title"] == "" {
		t.Fatalf("Details[title] 应非空: %v", de.Details)
	}
}

func TestNormalize_LengthByRune(t *testing.T) {
	in := validInput()
	in.Title = strings.Repeat("汉", 200)
	if _, err := in.Normalize(); err != nil {
		t.Fatalf("200 个汉字应通过: %v", err)
	}
	in.Title = strings.Repeat("汉", 201)
	de := asValidation(t, mustErr(t, in))
	if de.Details["title"] == "" {
		t.Fatalf("201 个汉字应被拒绝: %v", de.Details)
	}
}

func mustErr(t *testing.T, in CreateTicketInput) error {
	t.Helper()
	_, err := in.Normalize()
	if err == nil {
		t.Fatal("期望报错，实际通过")
	}
	return err
}

func TestNormalize_MultipleFieldErrors(t *testing.T) {
	in := CreateTicketInput{Title: " ", Description: "", CustomerName: " ", CustomerContact: "", Priority: "x"}
	de := asValidation(t, mustErr(t, in))
	for _, f := range []string{"title", "description", "customer_name", "customer_contact", "priority"} {
		if de.Details[f] == "" {
			t.Errorf("Details 缺少字段 %s: %v", f, de.Details)
		}
	}
}

func TestNormalize_Priority(t *testing.T) {
	for _, p := range []Priority{"critical", " high ", "", "HIGH"} {
		in := validInput()
		in.Priority = p
		de := asValidation(t, mustErr(t, in))
		if de.Details["priority"] == "" {
			t.Errorf("priority=%q 应被拒绝", p)
		}
	}
	for _, p := range []Priority{PriorityUrgent, PriorityHigh, PriorityNormal, PriorityLow} {
		in := validInput()
		in.Priority = p
		if _, err := in.Normalize(); err != nil {
			t.Errorf("priority=%q 应通过: %v", p, err)
		}
	}
}

func TestNormalize_FieldLimits(t *testing.T) {
	cases := []struct {
		field string
		max   int
		set   func(*CreateTicketInput, string)
	}{
		{"title", 200, func(i *CreateTicketInput, v string) { i.Title = v }},
		{"description", 5000, func(i *CreateTicketInput, v string) { i.Description = v }},
		{"customer_name", 100, func(i *CreateTicketInput, v string) { i.CustomerName = v }},
		{"customer_contact", 200, func(i *CreateTicketInput, v string) { i.CustomerContact = v }},
	}
	for _, c := range cases {
		in := validInput()
		c.set(&in, strings.Repeat("a", c.max))
		if _, err := in.Normalize(); err != nil {
			t.Errorf("%s 恰好 %d 字符应通过: %v", c.field, c.max, err)
		}
		c.set(&in, strings.Repeat("a", c.max+1))
		de := asValidation(t, mustErr(t, in))
		if de.Details[c.field] == "" || len(de.Details) != 1 {
			t.Errorf("%s 超长应只报该字段: %v", c.field, de.Details)
		}
	}
}

func TestNormalizeComment(t *testing.T) {
	if v, err := NormalizeComment("  你好  "); err != nil || v != "你好" {
		t.Fatalf("got %q, %v", v, err)
	}
	if _, err := NormalizeComment("   "); err == nil {
		t.Fatal("全空白评论应被拒绝")
	}
	if _, err := NormalizeComment(strings.Repeat("汉", 2000)); err != nil {
		t.Fatalf("2000 字符应通过: %v", err)
	}
	de := asValidation(t, func() error { _, e := NormalizeComment(strings.Repeat("汉", 2001)); return e }())
	if de.Details["body"] == "" {
		t.Fatalf("Details[body] 应非空: %v", de.Details)
	}
}

func TestNormalizeNote(t *testing.T) {
	if v, err := NormalizeNote("   "); err != nil || v != "" {
		t.Fatalf("空备注应通过: %q, %v", v, err)
	}
	if v, err := NormalizeNote(" 已处理 "); err != nil || v != "已处理" {
		t.Fatalf("got %q, %v", v, err)
	}
	if _, err := NormalizeNote(strings.Repeat("汉", 500)); err != nil {
		t.Fatalf("500 字符应通过: %v", err)
	}
	de := asValidation(t, func() error { _, e := NormalizeNote(strings.Repeat("汉", 501)); return e }())
	if de.Details["note"] == "" {
		t.Fatalf("Details[note] 应非空: %v", de.Details)
	}
}

func TestEnumParse(t *testing.T) {
	if r, ok := ParseRole("agent"); !ok || r != RoleAgent {
		t.Error("ParseRole(agent)")
	}
	if _, ok := ParseRole("Agent"); ok {
		t.Error("ParseRole 不应忽略大小写")
	}
	if s, ok := ParseStatus("in_progress"); !ok || s != StatusInProgress {
		t.Error("ParseStatus(in_progress)")
	}
	if _, ok := ParseStatus(" open"); ok {
		t.Error("ParseStatus 不应 trim")
	}
	if p, ok := ParsePriority("urgent"); !ok || p != PriorityUrgent {
		t.Error("ParsePriority(urgent)")
	}
	if _, ok := ParsePriority("critical"); ok {
		t.Error("ParsePriority(critical) 应失败")
	}
	for _, a := range append([]Action{ActionView}, ActionsForUI...) {
		if got, ok := ParseAction(string(a)); !ok || got != a {
			t.Errorf("ParseAction(%s)", a)
		}
	}
	if _, ok := ParseAction("delete"); ok {
		t.Error("ParseAction(delete) 应失败")
	}
}
