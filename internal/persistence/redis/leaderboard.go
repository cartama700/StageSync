// Package redis — Redis 기반 persistence 구현.
// 현재 ランキング (Phase 7) 만 사용. Redis 의 ZSET 을 레이블당 하나씩 둠.
package redis

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
)

// Leaderboard — Redis ZSET 기반 랭킹 저장소.
// 각 이벤트마다 키: `ranking:event:{eventID}`. 멤버=playerID, 스코어=누적 점수.
// 동점 처리: ZSET 기본 (lex 사전순). inmem 구현과 일치하도록 ZREVRANGEBYSCORE 대신 ZREVRANGE 사용.
type Leaderboard struct {
	client *goredis.Client
}

// NewLeaderboard — 이미 연결된 goredis.Client 를 받아 래핑.
func NewLeaderboard(client *goredis.Client) *Leaderboard {
	return &Leaderboard{client: client}
}

// key — ZSET 키 이름 규칙.
func key(eventID string) string {
	return "ranking:event:" + eventID
}

// IncrBy — ZINCRBY 로 원자적 증분 + 새 총점 반환.
func (l *Leaderboard) IncrBy(ctx context.Context, eventID, playerID string, delta int64) (int64, error) {
	newScore, err := l.client.ZIncrBy(ctx, key(eventID), float64(delta), playerID).Result()
	if err != nil {
		return 0, fmt.Errorf("ZINCRBY: %w", err)
	}
	return int64(newScore), nil
}

// Top — ZREVRANGE 0..n-1 WITHSCORES.
func (l *Leaderboard) Top(ctx context.Context, eventID string, n int) ([]domain.Entry, error) {
	if n <= 0 {
		return nil, nil
	}
	res, err := l.client.ZRevRangeWithScores(ctx, key(eventID), 0, int64(n-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("ZREVRANGE: %w", err)
	}
	out := make([]domain.Entry, len(res))
	for i, z := range res {
		pid, _ := z.Member.(string) // ZSET 멤버는 string
		out[i] = domain.Entry{
			PlayerID: pid,
			Score:    int64(z.Score),
			Rank:     i + 1,
		}
	}
	return out, nil
}

// Rank — ZREVRANK + ZSCORE 조합.
// 멤버 없으면 ErrPlayerNotRanked.
func (l *Leaderboard) Rank(ctx context.Context, eventID, playerID string) (*domain.Entry, error) {
	k := key(eventID)
	rank, err := l.client.ZRevRank(ctx, k, playerID).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, domain.ErrPlayerNotRanked
		}
		return nil, fmt.Errorf("ZREVRANK: %w", err)
	}
	score, err := l.client.ZScore(ctx, k, playerID).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, domain.ErrPlayerNotRanked
		}
		return nil, fmt.Errorf("ZSCORE: %w", err)
	}
	return &domain.Entry{
		PlayerID: playerID,
		Score:    int64(score),
		Rank:     int(rank) + 1, // Redis 는 0-based, 도메인은 1-based.
	}, nil
}

// Around — 본인 ±radius.
// ZREVRANK 로 인덱스 구한 뒤 ZREVRANGE [idx-radius, idx+radius] WITHSCORES.
func (l *Leaderboard) Around(ctx context.Context, eventID, playerID string, radius int) ([]domain.Entry, error) {
	k := key(eventID)
	rank, err := l.client.ZRevRank(ctx, k, playerID).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, domain.ErrPlayerNotRanked
		}
		return nil, fmt.Errorf("ZREVRANK: %w", err)
	}
	start := rank - int64(radius)
	if start < 0 {
		start = 0
	}
	end := rank + int64(radius)

	res, err := l.client.ZRevRangeWithScores(ctx, k, start, end).Result()
	if err != nil {
		return nil, fmt.Errorf("ZREVRANGE: %w", err)
	}
	out := make([]domain.Entry, len(res))
	for i, z := range res {
		pid, _ := z.Member.(string)
		out[i] = domain.Entry{
			PlayerID: pid,
			Score:    int64(z.Score),
			Rank:     int(start) + i + 1,
		}
	}
	return out, nil
}
