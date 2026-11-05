package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"ticketdesk/internal/domain"
)

// maxBody 是请求体上限（WF§17.2：64KB）。
const maxBody = 64 << 10

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// readJSON 把请求体解析到 dst。未知字段静默忽略（不开 DisallowUnknownFields），
// 这样客户端多传的 role、actor_id 等不会报错也不会生效（WF§2）。
// 空请求体按零值处理，让缺失字段交给 domain 校验，得到字段级错误。
func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return domain.ValidationError(map[string]string{
			"body": "请求体不是合法的 JSON，或超过 64KB",
		})
	}
	return nil
}
