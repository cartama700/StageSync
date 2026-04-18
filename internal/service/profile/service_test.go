// package profile_test — 외부 테스트 패키지 (외부 API 만 사용).
package profile_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	profilesvc "github.com/kimsehoon/stagesync/internal/service/profile"
)

// newService — 테스트용 Service + inmem repo 조립 헬퍼.
func newService(t *testing.T) *profilesvc.Service {
	t.Helper()
	return profilesvc.NewService(inmem.NewProfileRepo())
}

func TestService_GetProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seed     []string // 사전 생성할 ID 들
		queryID  string
		wantErr  error
		wantName string
	}{
		{
			name:     "found",
			seed:     []string{"p1"},
			queryID:  "p1",
			wantName: "player-p1",
		},
		{
			name:    "not found empty repo",
			queryID: "ghost",
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "not found different id",
			seed:    []string{"p1", "p2"},
			queryID: "p3",
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t)
			ctx := context.Background()
			for _, id := range tc.seed {
				_, err := svc.CreateProfile(ctx, id, "player-"+id)
				require.NoError(t, err)
			}

			got, err := svc.GetProfile(ctx, tc.queryID)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.queryID, got.ID)
			require.Equal(t, tc.wantName, got.Name)
		})
	}
}

func TestService_CreateProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    []string
		id      string
		pName   string
		wantErr error
	}{
		{name: "new profile", id: "p1", pName: "sekai"},
		{
			name:    "duplicate",
			seed:    []string{"p1"},
			id:      "p1",
			pName:   "other",
			wantErr: domain.ErrAlreadyExists,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t)
			ctx := context.Background()
			for _, id := range tc.seed {
				_, err := svc.CreateProfile(ctx, id, "seed-"+id)
				require.NoError(t, err)
			}

			p, err := svc.CreateProfile(ctx, tc.id, tc.pName)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.id, p.ID)
			require.Equal(t, tc.pName, p.Name)
			require.False(t, p.CreatedAt.IsZero(), "CreatedAt must be set")
		})
	}
}
