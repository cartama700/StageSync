// package profile_test — 외부 테스트 패키지 (external test package).
// service/profile 의 public API 만 사용 → 고객 시점 테스트 보장.
package profile_test

import (
	"context"
	"errors"
	"testing"

	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	profilesvc "github.com/kimsehoon/stagesync/internal/service/profile"
)

// TestCreateAndGet — 생성 후 조회 기본 흐름.
func TestCreateAndGet(t *testing.T) {
	repo := inmem.NewProfileRepo()
	svc := profilesvc.NewService(repo)
	ctx := context.Background()

	created, err := svc.CreateProfile(ctx, "p1", "sekai")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Name != "sekai" {
		t.Errorf("name: got=%q want=%q", created.Name, "sekai")
	}

	got, err := svc.GetProfile(ctx, "p1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "p1" {
		t.Errorf("id: got=%q want=%q", got.ID, "p1")
	}
}

// TestGet_NotFound — 없는 ID 조회 시 domain.ErrNotFound 를 errors.Is 로 판별.
func TestGet_NotFound(t *testing.T) {
	repo := inmem.NewProfileRepo()
	svc := profilesvc.NewService(repo)

	_, err := svc.GetProfile(context.Background(), "nope")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err: got=%v, want wrap of ErrNotFound", err)
	}
}

// TestCreate_Duplicate — 중복 생성 시 domain.ErrAlreadyExists.
func TestCreate_Duplicate(t *testing.T) {
	repo := inmem.NewProfileRepo()
	svc := profilesvc.NewService(repo)
	ctx := context.Background()

	if _, err := svc.CreateProfile(ctx, "p1", "sekai"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.CreateProfile(ctx, "p1", "sekai2")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("err: got=%v, want wrap of ErrAlreadyExists", err)
	}
}
