package httpapi

import (
	"errors"
	"log"
	"net/http"

	"ticketdesk/internal/domain"
)

// statusByCode 是 WF§9 错误码表。httpapi 的映射表测试会遍历 domain.AllCodes，
// 新增 code 却漏映射时测试失败。
var statusByCode = map[domain.Code]int{
	domain.CodeValidationError:   http.StatusBadRequest,
	domain.CodeUnauthenticated:   http.StatusUnauthorized,
	domain.CodeForbidden:         http.StatusForbidden,
	domain.CodeNotAssignee:       http.StatusForbidden,
	domain.CodeTicketNotFound:    http.StatusNotFound,
	domain.CodeAlreadyClaimed:    http.StatusConflict,
	domain.CodeInvalidTransition: http.StatusConflict,
	domain.CodeInvalidState:      http.StatusConflict,
	domain.CodeSameAssignee:      http.StatusConflict,
	domain.CodeInvalidAssignee:   http.StatusUnprocessableEntity,
	domain.CodeInternal:          http.StatusInternalServerError,
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    domain.Code       `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

// writeError 按 WF§9 输出统一格式。非 *domain.Error 记日志后返回 500 INTERNAL，
// 响应里不含原始错误文本。
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var de *domain.Error
	if !errors.As(err, &de) {
		log.Printf("internal error: %s %s: %v", r.Method, r.URL.Path, err)
		de = domain.ErrInternal()
	}
	status, ok := statusByCode[de.Code]
	if !ok {
		status = http.StatusInternalServerError
	}
	details := de.Details
	if details == nil {
		details = map[string]string{}
	}
	writeJSON(w, status, errorBody{errorDetail{de.Code, de.Message, details}})
}
