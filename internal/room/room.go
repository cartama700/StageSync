package room

import (
	"sync"

	"github.com/kimsehoon/stagesync/api/proto/roompb"
)

// Member — 접속 중인 플레이어의 최신 상태.
type Member struct {
	PlayerID string
	X, Y     float64
}

// Room — 메모리 상의 스레드 안전 플레이어 레지스트리.
// Diarkis ルーム 기능의 최소 모사 (Phase 1 은 최신 위치만 추적).
// 브로드캐스트는 후속 Phase 에서 추가.
type Room struct {
	mu      sync.RWMutex
	members map[string]*Member
}

// NewRoom — 빈 Room 생성.
func NewRoom() *Room {
	return &Room{members: map[string]*Member{}}
}

// ApplyMove — 플레이어의 최신 위치를 upsert.
func (r *Room) ApplyMove(m *roompb.Move) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := m.GetPlayerId()
	member, ok := r.members[id]
	if !ok {
		member = &Member{PlayerID: id}
		r.members[id] = member
	}
	member.X = m.GetX()
	member.Y = m.GetY()
}

// Remove — 플레이어 제거 (WebSocket 연결 종료 시 호출).
func (r *Room) Remove(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.members, playerID)
}

// Size — 접속 중인 플레이어 수.
func (r *Room) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}
