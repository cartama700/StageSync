package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/kimsehoon/stagesync/internal/apperror"
	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
)

// ProfileService — 이 handler 가 요구하는 서비스 인터페이스 (consumer-defined).
type ProfileService interface {
	GetProfile(ctx context.Context, id string) (*domain.Profile, error)
	CreateProfile(ctx context.Context, id, name string) (*domain.Profile, error)
}

// ProfileHandler — プロフィール HTTP 핸들러 묶음.
type ProfileHandler struct {
	Service ProfileService
}

// Mount — プロフィール 도메인의 라우트를 부모 라우터에 등록.
// 도메인이 자기 URL prefix 를 책임지는 패턴 (main.go 에서 한 줄로 호출).
func (h *ProfileHandler) Mount(r chi.Router) {
	r.Get("/api/profile/{id}", h.Get)
	r.Post("/api/profile", h.Create)
}

// vldtr — 패키지 전역 validator (thread-safe, 재사용 권장).
// JSON 태그명을 필드명으로 쓰도록 커스터마이즈 → 에러 응답의 "field" 가 클라 JSON 과 일치.
var vldtr = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}()

// ----- DTO -----

// profileDTO — HTTP 경계 DTO. domain.Profile 와 분리.
type profileDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func toDTO(p *domain.Profile) profileDTO {
	return profileDTO{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt}
}

// createReq — POST /api/profile 요청 body.
// validator 태그로 필드별 제약 명시 — 길이·필수 여부 등.
type createReq struct {
	ID   string `json:"id"   validate:"required,min=1,max=64"`
	Name string `json:"name" validate:"required,min=1,max=128"`
}

// ----- 핸들러 -----

// Get — GET /api/profile/{id}.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		apperror.WriteJSON(w, apperror.Validation("id path param is required", nil))
		return
	}

	p, err := h.Service.GetProfile(r.Context(), id)
	if err != nil {
		// domain sentinel → apperror 로 변환 (endpoint 가 HTTP 맥락 담당).
		if errors.Is(err, domain.ErrNotFound) {
			apperror.WriteJSON(w, apperror.NotFound("profile", id))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("get profile", err))
		return
	}
	writeJSON(w, toDTO(p))
}

// Create — POST /api/profile.
func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation("invalid JSON body", nil))
		return
	}
	if err := vldtr.Struct(req); err != nil {
		apperror.WriteJSON(w, toValidationError(err))
		return
	}

	p, err := h.Service.CreateProfile(r.Context(), req.ID, req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			apperror.WriteJSON(w, apperror.Conflict("profile already exists"))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("create profile", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, toDTO(p))
}

// ----- validator 에러 → apperror 변환 -----

// toValidationError — validator 의 ValidationErrors 를 apperror.FieldError 배열로.
func toValidationError(err error) *apperror.Error {
	var verr validator.ValidationErrors
	if !errors.As(err, &verr) {
		return apperror.Validation("validation failed", nil)
	}
	fields := make([]apperror.FieldError, 0, len(verr))
	for _, fe := range verr {
		fields = append(fields, apperror.FieldError{
			Field:   fe.Field(),
			Tag:     fe.Tag(),
			Message: humanizeFieldError(fe),
		})
	}
	return apperror.Validation("input validation failed", fields)
}

// humanizeFieldError — validator 태그별 사람 친화 메시지.
func humanizeFieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "min":
		return fe.Field() + " is too short (min=" + fe.Param() + ")"
	case "max":
		return fe.Field() + " is too long (max=" + fe.Param() + ")"
	default:
		return fe.Field() + " is invalid (" + fe.Tag() + ")"
	}
}
