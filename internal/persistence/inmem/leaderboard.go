package inmem

import (
	"context"
	"sort"
	"sync"

	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
)

// Leaderboard — 메모리 기반 랭킹 저장소 (ranking.Store 및 event.LeaderboardWriter 만족).
// REDIS_ADDR 미설정 시 graceful fallback. 프로덕션 대체품 아님 — 테스트·개발 편의.
//
// 복잡도: 쓰기 O(1), 읽기 O(N log N) — 조회 때마다 정렬. 작은 이벤트 (수천명 이내) 용.
// 정렬: 점수 DESC, 동점은 playerID DESC — Redis `ZREVRANGE` 의 기본 동점 처리와 정확히 일치.
// (Redis ZSET 은 score 같을 때 lex 사전순 정렬 → ZREVRANGE 로 뒤집으면 lex DESC 가 됨)
type Leaderboard struct {
	mu     sync.RWMutex
	scores map[string]map[string]int64 // eventID -> playerID -> score
}

// NewLeaderboard — 빈 저장소 생성.
func NewLeaderboard() *Leaderboard {
	return &Leaderboard{scores: map[string]map[string]int64{}}
}

// IncrBy — (event, player) 점수 증분 + 누적 총점 반환.
func (l *Leaderboard) IncrBy(_ context.Context, eventID, playerID string, delta int64) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	inner, ok := l.scores[eventID]
	if !ok {
		inner = map[string]int64{}
		l.scores[eventID] = inner
	}
	inner[playerID] += delta
	return inner[playerID], nil
}

// Top — 점수 내림차순 상위 n 개.
func (l *Leaderboard) Top(_ context.Context, eventID string, n int) ([]domain.Entry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	sorted := l.sortedLocked(eventID)
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n], nil
}

// Rank — 단일 플레이어 순위.
func (l *Leaderboard) Rank(_ context.Context, eventID, playerID string) (*domain.Entry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	sorted := l.sortedLocked(eventID)
	for _, e := range sorted {
		if e.PlayerID == playerID {
			e := e // avoid loop var alias
			return &e, nil
		}
	}
	return nil, domain.ErrPlayerNotRanked
}

// Around — 본인 ±radius. Rank 1 쪽 또는 꼴찌 쪽 경계에 clamp.
func (l *Leaderboard) Around(_ context.Context, eventID, playerID string, radius int) ([]domain.Entry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	sorted := l.sortedLocked(eventID)
	idx := -1
	for i, e := range sorted {
		if e.PlayerID == playerID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, domain.ErrPlayerNotRanked
	}
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + radius + 1
	if end > len(sorted) {
		end = len(sorted)
	}
	out := make([]domain.Entry, end-start)
	copy(out, sorted[start:end])
	return out, nil
}

// sortedLocked — eventID 의 엔트리를 (점수 desc, playerID asc) 정렬해서 Rank 부여.
// 반드시 caller 가 lock 보유 상태에서 호출.
func (l *Leaderboard) sortedLocked(eventID string) []domain.Entry {
	inner, ok := l.scores[eventID]
	if !ok {
		return nil
	}
	entries := make([]domain.Entry, 0, len(inner))
	for p, s := range inner {
		entries = append(entries, domain.Entry{PlayerID: p, Score: s})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		// 동점은 playerID 역순 — Redis ZREVRANGE 와 일치.
		return entries[i].PlayerID > entries[j].PlayerID
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}
