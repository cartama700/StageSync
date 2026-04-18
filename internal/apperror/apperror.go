// Package apperror — 애플리케이션 전역 에러 타입.
// endpoint 레이어가 HTTP status 로 매핑하기 위해 타입 판별 (errors.As).
// domain sentinel 에러와 분리 — service·repo 는 domain 에러, endpoint 가 apperror 로 변환.
package apperror

import "fmt"

// Code — 에러 분류 코드. HTTP 응답의 "code" 필드로 노출 (클라가 분기 가능).
type Code string

const (
	CodeValidation Code = "VALIDATION_FAILED"
	CodeNotFound   Code = "NOT_FOUND"
	CodeConflict   Code = "CONFLICT"
	CodeInternal   Code = "INTERNAL"
)

// FieldError — 필드별 validation 실패 상세. JSON 응답의 "fields" 배열에 담김.
type FieldError struct {
	Field   string `json:"field"`   // 실패 필드명
	Tag     string `json:"tag"`     // validator 태그 (required / min / max 등)
	Message string `json:"message"` // 사람이 읽을 수 있는 설명
}

// Error — 애플리케이션 표준 에러. Go error 인터페이스 만족.
// Unwrap() 지원으로 errors.Is / errors.As 체인 정상 동작.
type Error struct {
	Code    Code         `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
	cause   error        // 원인 에러 (JSON 노출 X — 내부 로그용)
}

// Error — error 인터페이스 구현.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap — errors.Is / errors.As 가 체인 거슬러 올라갈 수 있게 함.
func (e *Error) Unwrap() error { return e.cause }

// ----- 생성 헬퍼 -----

// Validation — 입력 검증 실패.
func Validation(msg string, fields []FieldError) *Error {
	return &Error{Code: CodeValidation, Message: msg, Fields: fields}
}

// NotFound — 리소스 없음.
func NotFound(resource, id string) *Error {
	return &Error{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s not found: %s", resource, id),
	}
}

// Conflict — 중복·충돌 (duplicate key 등).
func Conflict(msg string) *Error {
	return &Error{Code: CodeConflict, Message: msg}
}

// Internal — 서버 내부 에러. cause 는 원인 체인 (%w 래핑 유지).
func Internal(msg string, cause error) *Error {
	return &Error{Code: CodeInternal, Message: msg, cause: cause}
}
