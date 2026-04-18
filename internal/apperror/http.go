package apperror

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// StatusOf — Error.Code → HTTP status 매핑.
// 분류 안 된 Code 는 500.
func StatusOf(e *Error) int {
	switch e.Code {
	case CodeValidation:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// WriteJSON — 에러 타입 판별하여 표준 JSON 응답.
// *Error 아닌 일반 error 도 받아서 500 Internal 로 감싸서 응답.
func WriteJSON(w http.ResponseWriter, err error) {
	var appErr *Error
	if errors.As(err, &appErr) {
		writeAppErr(w, appErr)
		return
	}
	// 분류 안 된 에러 — 원본은 내부 로그로만, 외부엔 일반화된 500.
	slog.Error("unclassified error", "err", err)
	writeAppErr(w, Internal("internal error", err))
}

// writeAppErr — HTTP 응답 실 직렬화.
func writeAppErr(w http.ResponseWriter, e *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(StatusOf(e))
	body := map[string]any{"error": e}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write error response", "err", err)
	}
}
