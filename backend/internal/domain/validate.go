package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// 字段长度上限（WF§6），按 Unicode 字符数计算。
const (
	MaxTitle           = 200
	MaxDescription     = 5000
	MaxCustomerName    = 100
	MaxCustomerContact = 200
	MaxCommentBody     = 2000
	MaxNote            = 500
)

type CreateTicketInput struct {
	Title           string
	Description     string
	CustomerName    string
	CustomerContact string
	Priority        Priority
}

// checkText 去首尾空白后校验字符数，返回规整后的值；失败时把原因写入 details[field]。
func checkText(details map[string]string, field, label, v string, min, max int) string {
	v = strings.TrimSpace(v)
	n := utf8.RuneCountInString(v)
	switch {
	case n < min:
		details[field] = label + "不能为空"
	case n > max:
		details[field] = fmt.Sprintf("%s不能超过 %d 个字符", label, max)
	}
	return v
}

// Normalize 去除文本字段首尾空白后校验（WF§6）。枚举值不做 trim，避免隐式纠错。
func (in CreateTicketInput) Normalize() (CreateTicketInput, error) {
	details := map[string]string{}
	out := CreateTicketInput{
		Title:           checkText(details, "title", "标题", in.Title, 1, MaxTitle),
		Description:     checkText(details, "description", "描述", in.Description, 1, MaxDescription),
		CustomerName:    checkText(details, "customer_name", "客户名称", in.CustomerName, 1, MaxCustomerName),
		CustomerContact: checkText(details, "customer_contact", "联系方式", in.CustomerContact, 1, MaxCustomerContact),
		Priority:        in.Priority,
	}
	if !in.Priority.Valid() {
		details["priority"] = "优先级必须是 urgent、high、normal、low 之一"
	}
	if len(details) > 0 {
		return CreateTicketInput{}, ValidationError(details)
	}
	return out, nil
}

// NormalizeComment 校验评论正文：去空白后 1～2000 字符。
func NormalizeComment(body string) (string, error) {
	details := map[string]string{}
	v := checkText(details, "body", "评论内容", body, 1, MaxCommentBody)
	if len(details) > 0 {
		return "", ValidationError(details)
	}
	return v, nil
}

// NormalizeNote 校验状态流转备注：去空白后 0～500 字符，可为空。
func NormalizeNote(note string) (string, error) {
	details := map[string]string{}
	v := checkText(details, "note", "备注", note, 0, MaxNote)
	if len(details) > 0 {
		return "", ValidationError(details)
	}
	return v, nil
}
