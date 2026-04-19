package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/auth"
)

// fixedClock — 테스트용 결정적 시계.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// TestIssuer_Issue_RoundtripOK —
// Issuer 로 발급한 토큰을 동일 secret 의 Validator 로 검증하면 성공 + player_id 복원.
func TestIssuer_Issue_RoundtripOK(t *testing.T) {
	t.Parallel()

	secret := "test-secret-32-bytes-at-least-ok"
	now := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

	issuer, err := auth.NewIssuer(secret, 15*time.Minute, auth.WithClock(fixedClock(now)))
	require.NoError(t, err)

	token, exp, err := issuer.Issue("p1")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, now.Add(15*time.Minute), exp)

	v := auth.NewValidator(secret, auth.WithValidatorClock(fixedClock(now)))
	require.NotNil(t, v)

	claims, err := v.Validate(token)
	require.NoError(t, err)
	require.Equal(t, "p1", claims.PlayerID)
}

// TestValidator_Expired — ttl 초과 시 ErrInvalidToken.
func TestValidator_Expired(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	issueAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	issuer, err := auth.NewIssuer(secret, 1*time.Second, auth.WithClock(fixedClock(issueAt)))
	require.NoError(t, err)

	token, _, err := issuer.Issue("p1")
	require.NoError(t, err)

	// 검증 시각은 2분 후 → 만료.
	v := auth.NewValidator(secret, auth.WithValidatorClock(fixedClock(issueAt.Add(2*time.Minute))))
	_, err = v.Validate(token)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestValidator_WrongSecret — 다른 secret 으로 검증 시 ErrInvalidToken.
func TestValidator_WrongSecret(t *testing.T) {
	t.Parallel()

	issuer, err := auth.NewIssuer("secret-A", 5*time.Minute)
	require.NoError(t, err)
	token, _, err := issuer.Issue("p1")
	require.NoError(t, err)

	v := auth.NewValidator("secret-B")
	_, err = v.Validate(token)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestValidator_EmptyToken — 빈 문자열 → ErrInvalidToken.
func TestValidator_EmptyToken(t *testing.T) {
	t.Parallel()
	v := auth.NewValidator("s")
	_, err := v.Validate("")
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestValidator_Malformed — JWT 포맷 아닌 문자열 → ErrInvalidToken.
func TestValidator_Malformed(t *testing.T) {
	t.Parallel()
	v := auth.NewValidator("s")
	_, err := v.Validate("not-a-jwt")
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestNewValidator_EmptySecret_ReturnsNil —
// secret 이 빈 문자열이면 nil Validator → 미들웨어는 이를 "auth 비활성" 으로 해석.
func TestNewValidator_EmptySecret_ReturnsNil(t *testing.T) {
	t.Parallel()
	require.Nil(t, auth.NewValidator(""))
}

// TestNewIssuer_EmptySecretRejected — Issuer 는 빈 secret 허용 안 함.
func TestNewIssuer_EmptySecretRejected(t *testing.T) {
	t.Parallel()
	_, err := auth.NewIssuer("", 1*time.Minute)
	require.Error(t, err)
}

// TestIssuer_EmptyPlayerIDRejected — player_id 빈 문자열은 발급 거절.
func TestIssuer_EmptyPlayerIDRejected(t *testing.T) {
	t.Parallel()
	issuer, err := auth.NewIssuer("s", 1*time.Minute)
	require.NoError(t, err)
	_, _, err = issuer.Issue("")
	require.Error(t, err)
}

// TestCtxHelpers — InjectClaims / PlayerIDFrom 왕복.
func TestCtxHelpers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// 없으면 빈값 + false.
	pid, ok := auth.PlayerIDFrom(ctx)
	require.False(t, ok)
	require.Empty(t, pid)

	// 주입 후 꺼내기.
	claims := &auth.Claims{PlayerID: "p42"}
	ctx2 := auth.InjectClaims(ctx, claims)
	pid, ok = auth.PlayerIDFrom(ctx2)
	require.True(t, ok)
	require.Equal(t, "p42", pid)

	got, ok := auth.ClaimsFrom(ctx2)
	require.True(t, ok)
	require.Same(t, claims, got)
}
