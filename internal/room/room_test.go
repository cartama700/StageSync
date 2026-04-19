package room_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/api/proto/roompb"
	"github.com/kimsehoon/stagesync/internal/room"
)

// TestRoom_NewRoom_Empty — 새 Room 은 비어 있다.
func TestRoom_NewRoom_Empty(t *testing.T) {
	t.Parallel()
	r := room.NewRoom()
	require.Equal(t, 0, r.Size())
}

// TestRoom_ApplyMove_Insert — 처음 보는 플레이어는 upsert.
func TestRoom_ApplyMove_Insert(t *testing.T) {
	t.Parallel()
	r := room.NewRoom()

	r.ApplyMove(&roompb.Move{PlayerId: "p1", X: 1.5, Y: 2.5})
	require.Equal(t, 1, r.Size())

	r.ApplyMove(&roompb.Move{PlayerId: "p2", X: 3.0, Y: 4.0})
	require.Equal(t, 2, r.Size())
}

// TestRoom_ApplyMove_Update — 같은 플레이어의 두 번째 move 는 덮어쓰기.
func TestRoom_ApplyMove_Update(t *testing.T) {
	t.Parallel()
	r := room.NewRoom()

	r.ApplyMove(&roompb.Move{PlayerId: "p1", X: 1, Y: 1})
	r.ApplyMove(&roompb.Move{PlayerId: "p1", X: 99, Y: 100})

	require.Equal(t, 1, r.Size(), "동일 PlayerId 는 새 엔트리가 아니라 update")
}

// TestRoom_Remove — 플레이어 삭제 후 Size 감소.
func TestRoom_Remove(t *testing.T) {
	t.Parallel()
	r := room.NewRoom()

	r.ApplyMove(&roompb.Move{PlayerId: "p1"})
	r.ApplyMove(&roompb.Move{PlayerId: "p2"})
	require.Equal(t, 2, r.Size())

	r.Remove("p1")
	require.Equal(t, 1, r.Size())

	r.Remove("p2")
	require.Equal(t, 0, r.Size())
}

// TestRoom_Remove_Missing — 존재하지 않는 플레이어 삭제는 no-op.
func TestRoom_Remove_Missing(t *testing.T) {
	t.Parallel()
	r := room.NewRoom()
	r.Remove("does-not-exist")
	require.Equal(t, 0, r.Size())
}

// TestRoom_ConcurrentApplyMove — 고루틴 1000개가 동시에 upsert 해도
// 데이터 레이스 없이 최종 Size 가 unique PlayerId 수와 일치.
// -race 플래그와 함께 실행하면 의미 있음.
func TestRoom_ConcurrentApplyMove(t *testing.T) {
	t.Parallel()
	r := room.NewRoom()

	const N = 1000
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			// 1/10 확률로 같은 ID — update 경로도 타게.
			id := "player"
			if i%10 != 0 {
				id = "player-" + strconv.Itoa(i)
			}
			r.ApplyMove(&roompb.Move{PlayerId: id, X: float64(i)})
		}()
	}
	wg.Wait()

	// i%10==0 은 전부 "player" 하나로 수렴 → unique id 수 = 1 + (N - N/10).
	expected := 1 + (N - N/10)
	require.Equal(t, expected, r.Size())
}
