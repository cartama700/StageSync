package endpoint

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON — endpoint 패키지 공용 JSON 응답 헬퍼.
// Content-Type 설정 + 인코딩. 인코딩 실패는 로그만 (응답 헤더 이미 나간 상태라 복구 불가).
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("json encode", "err", err)
	}
}
