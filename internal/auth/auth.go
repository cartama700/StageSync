// Package auth — JWT 기반 認証 · 認可 프리미티브.
//
// 설계 원칙:
//   - 핸들러는 URL 파라미터·body 의 player_id 를 **신뢰하지 않고** `PlayerIDFrom(ctx)` 로만 취득
//   - `AuthSecret` 이 빈 문자열이면 미들웨어가 **no-op** (로컬 개발 편의 · 기존 테스트 호환)
//   - JWT 는 HS256 (HMAC-SHA256) — 단일 프로세스 배포에 충분. 프로덕션 MSA 면 RS256 + 공개키 배포로 전환.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — JWT payload. `jwt.RegisteredClaims` 로 exp · iat · nbf · iss · jti 표준.
type Claims struct {
	PlayerID string `json:"player_id"`
	jwt.RegisteredClaims
}

// ----- Issuer -----

// Issuer — JWT 를 발급. 개발용 `/api/auth/login` 에서 사용.
// 프로덕션이라면 외부 IdP (Auth0 · Firebase Auth · 자체 SSO) 가 대체.
type Issuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
	issuer string
}

// NewIssuer — secret 과 TTL 로 Issuer 생성. secret 은 빈 값 금지.
func NewIssuer(secret string, ttl time.Duration, opts ...IssuerOption) (*Issuer, error) {
	if secret == "" {
		return nil, errors.New("auth: secret must not be empty")
	}
	if ttl <= 0 {
		return nil, errors.New("auth: ttl must be > 0")
	}
	is := &Issuer{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
		issuer: "stagesync",
	}
	for _, o := range opts {
		o(is)
	}
	return is, nil
}

// IssuerOption — 테스트용 clock 주입 등.
type IssuerOption func(*Issuer)

// WithClock — 시계 함수 주입 (테스트 결정성).
func WithClock(fn func() time.Time) IssuerOption {
	return func(i *Issuer) { i.now = fn }
}

// WithIssuer — `iss` 클레임 커스터마이즈.
func WithIssuer(name string) IssuerOption {
	return func(i *Issuer) { i.issuer = name }
}

// Issue — playerID 에 대한 JWT 를 발급.
func (i *Issuer) Issue(playerID string) (string, time.Time, error) {
	if playerID == "" {
		return "", time.Time{}, errors.New("auth: playerID must not be empty")
	}
	now := i.now()
	exp := now.Add(i.ttl)
	claims := Claims{
		PlayerID: playerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   playerID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

// ----- Validator -----

// Validator — JWT 검증. 미들웨어가 요청마다 호출.
type Validator struct {
	secret []byte
	now    func() time.Time
	issuer string
}

// NewValidator — secret 만으로 생성 (기본 issuer / 시계).
// secret 이 빈 문자열이면 nil + nil — 호출자 (미들웨어) 가 이를 "auth 비활성" 시그널로 해석.
func NewValidator(secret string, opts ...ValidatorOption) *Validator {
	if secret == "" {
		return nil
	}
	v := &Validator{
		secret: []byte(secret),
		now:    time.Now,
		issuer: "stagesync",
	}
	for _, o := range opts {
		o(v)
	}
	return v
}

// ValidatorOption — 테스트용 옵션.
type ValidatorOption func(*Validator)

// WithValidatorClock — 시계 함수 주입.
func WithValidatorClock(fn func() time.Time) ValidatorOption {
	return func(v *Validator) { v.now = fn }
}

// WithValidatorIssuer — expected `iss` 클레임 설정.
func WithValidatorIssuer(name string) ValidatorOption {
	return func(v *Validator) { v.issuer = name }
}

// ErrInvalidToken — 검증 실패 총칭 (서명 불일치 · 만료 · 잘못된 클레임 등).
// 호출자는 errors.Is 로 판별 → 401 응답.
var ErrInvalidToken = errors.New("auth: invalid token")

// Validate — 토큰 문자열을 검증하고 Claims 반환.
// 만료·서명·issuer 불일치는 모두 ErrInvalidToken 으로 래핑.
func (v *Validator) Validate(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithTimeFunc(v.now),
	)
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		return v.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.PlayerID == "" {
		return nil, fmt.Errorf("%w: empty player_id", ErrInvalidToken)
	}
	return claims, nil
}

// ----- context helpers -----

// ctxKey — 패키지 외부에서 오염시킬 수 없는 private key.
type ctxKey struct{}

// InjectClaims — ctx 에 Claims 를 주입. 미들웨어가 호출.
func InjectClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// ClaimsFrom — ctx 에서 Claims 반환. 미설정 시 nil + false.
func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok
}

// PlayerIDFrom — ctx 에서 authenticated player_id 반환.
// 미설정이거나 ctx 에 Claims 가 없으면 빈 문자열 + false.
//
// **핸들러는 URL 파라미터 · body 의 player_id 를 신뢰하지 말고
// 이 함수의 반환값을 truth 로 사용해야 함.**
func PlayerIDFrom(ctx context.Context) (string, bool) {
	c, ok := ClaimsFrom(ctx)
	if !ok {
		return "", false
	}
	return c.PlayerID, true
}
