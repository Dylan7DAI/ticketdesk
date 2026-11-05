package domain

// Code 的取值与 WF§9 错误码表一一对应。
type Code string

const (
	CodeValidationError   Code = "VALIDATION_ERROR"
	CodeUnauthenticated   Code = "UNAUTHENTICATED"
	CodeForbidden         Code = "FORBIDDEN"
	CodeNotAssignee       Code = "NOT_ASSIGNEE"
	CodeTicketNotFound    Code = "TICKET_NOT_FOUND"
	CodeAlreadyClaimed    Code = "ALREADY_CLAIMED"
	CodeInvalidTransition Code = "INVALID_TRANSITION"
	CodeInvalidState      Code = "INVALID_STATE"
	CodeSameAssignee      Code = "SAME_ASSIGNEE"
	CodeInvalidAssignee   Code = "INVALID_ASSIGNEE"
	CodeInternal          Code = "INTERNAL"
)

// AllCodes 列出全部错误码，供 httpapi 的映射表测试遍历，防止新增 code 时漏映射。
var AllCodes = []Code{
	CodeValidationError, CodeUnauthenticated, CodeForbidden, CodeNotAssignee,
	CodeTicketNotFound, CodeAlreadyClaimed, CodeInvalidTransition, CodeInvalidState,
	CodeSameAssignee, CodeInvalidAssignee, CodeInternal,
}

// Error 是业务错误。函数返回 error 类型时，成功必须写 return nil，
// 不能返回值为 nil 的 *Error 变量（typed nil 会让 err != nil 为真）。
type Error struct {
	Code    Code
	Message string
	Details map[string]string
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Message }

func newErr(c Code, msg string) *Error { return &Error{Code: c, Message: msg} }

// ValidationError 的 details 为 {字段: 原因}。
func ValidationError(details map[string]string) *Error {
	return &Error{Code: CodeValidationError, Message: "请求参数有误", Details: details}
}

func ErrUnauthenticated() *Error {
	return newErr(CodeUnauthenticated, "请先选择有效的用户身份")
}

func ErrForbidden() *Error { return newErr(CodeForbidden, "当前角色无权执行此操作") }

func ErrNotAssignee() *Error {
	return newErr(CodeNotAssignee, "只有工单处理人或主管可以执行此操作")
}

func ErrTicketNotFound() *Error { return newErr(CodeTicketNotFound, "工单不存在") }

func ErrAlreadyClaimed() *Error { return newErr(CodeAlreadyClaimed, "工单已被其他客服领取") }

func ErrInvalidTransition() *Error {
	return newErr(CodeInvalidTransition, "当前状态下不能执行该状态流转")
}

func ErrInvalidState() *Error {
	return newErr(CodeInvalidState, "当前工单状态不允许此操作")
}

func ErrSameAssignee() *Error {
	return newErr(CodeSameAssignee, "改派目标与当前处理人相同")
}

func ErrInvalidAssignee() *Error {
	return newErr(CodeInvalidAssignee, "目标用户不存在或不是客服")
}

func ErrInternal() *Error { return newErr(CodeInternal, "服务器内部错误") }
