// cmd/battlebench — Phase 19 HP 데드락 랩의 실측 CLI.
//
// 한 target_player 에 대해 N 개 고루틴이 동시에 ApplyDamage 를 호출 →
// 구현별 (v1-naive / v2-queue) 처리량 · 에러율 · p50/p95/p99 latency 를 측정.
//
// **실 MySQL 권장**: inmem 은 락 경합 재현 불가.
// 전제 조건: `make dev-up` 또는 `docker compose up mysql` 로 MySQL 실행 중 + goose 마이그레이션 적용 완료.
//
// 사용 예:
//
//	# v1-naive (FOR UPDATE 기반) — 락 경합 발생 기대.
//	MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/stagesync?parseTime=true' \
//	  go run ./cmd/battlebench -impl=naive -n=100 -target=boss-1
//
//	# v2-queue — playerID 별 직렬화로 lock wait timeout 0 기대.
//	MYSQL_DSN='...' go run ./cmd/battlebench -impl=queue -n=100 -target=boss-1
//
// 두 번 실행해서 수치를 `docs/BENCHMARKS.md` 의 Phase 19 표에 채워 넣는 용도.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kimsehoon/stagesync/internal/persistence/mysql"
	battlesvc "github.com/kimsehoon/stagesync/internal/service/battle"
)

func main() {
	var (
		dsn     = flag.String("dsn", os.Getenv("MYSQL_DSN"), "MySQL DSN (기본: MYSQL_DSN env)")
		impl    = flag.String("impl", "naive", "구현 선택: naive | queue")
		target  = flag.String("target", "boss-1", "공격 타겟 playerID")
		n       = flag.Int("n", 100, "동시 공격 수")
		damage  = flag.Int("damage", 10, "공격당 데미지")
		hpInit  = flag.Int("hp", 1_000_000, "초기 HP 설정값")
		timeout = flag.Duration("timeout", 30*time.Second, "전체 벤치 타임아웃")
	)
	flag.Parse()

	if *dsn == "" {
		log.Fatal("MYSQL_DSN 이 필요합니다. env 로 설정하거나 -dsn 플래그 사용")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	db, err := mysql.Open(ctx, *dsn)
	if err != nil {
		log.Fatalf("mysql open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := mysql.NewBattleRepo(db)
	if err := repo.Reset(ctx, *target, *hpInit); err != nil {
		log.Fatalf("reset target hp: %v", err)
	}

	applier := battlesvc.Build(battlesvc.Implementation(*impl), repo)
	fmt.Printf("=== Phase 19 bench: impl=%s target=%s n=%d damage=%d hp_init=%d ===\n",
		*impl, *target, *n, *damage, *hpInit)

	results := runBench(ctx, applier, *target, *n, *damage)
	printReport(results, *impl)

	// 최종 HP 확인 — 누적 일치성 체크.
	finalHP, err := repo.Get(context.Background(), *target)
	if err != nil {
		log.Printf("warning: get final hp: %v", err)
	} else {
		expected := *hpInit - (*damage)*results.ok
		fmt.Printf("final_hp=%d, expected=%d (hp_init - damage*ok), delta=%d\n",
			finalHP.HP, expected, finalHP.HP-expected)
	}
}

// result — 한 고루틴의 실행 결과.
type result struct {
	dur time.Duration
	err error
}

// summary — 집계 통계.
type summary struct {
	total int
	ok    int
	fail  int
	errs  map[string]int

	p50, p95, p99, min, max, avg time.Duration
	throughputRPS                float64
}

// runBench — N 고루틴 동시 실행.
func runBench(ctx context.Context, applier battlesvc.Applier, target string, n, damage int) summary {
	results := make([]result, n)
	var ok, fail int32

	var wg sync.WaitGroup
	wg.Add(n)

	start := time.Now()
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			t0 := time.Now()
			_, err := applier.Apply(ctx, target, damage)
			dur := time.Since(t0)
			results[i] = result{dur: dur, err: err}
			if err != nil {
				atomic.AddInt32(&fail, 1)
			} else {
				atomic.AddInt32(&ok, 1)
			}
		}()
	}
	wg.Wait()
	total := time.Since(start)

	// 집계.
	s := summary{total: n, ok: int(ok), fail: int(fail), errs: map[string]int{}}
	durs := make([]time.Duration, 0, n)
	for _, r := range results {
		if r.err != nil {
			s.errs[r.err.Error()]++
			continue
		}
		durs = append(durs, r.dur)
	}
	if len(durs) > 0 {
		sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
		s.min = durs[0]
		s.max = durs[len(durs)-1]
		s.p50 = durs[len(durs)*50/100]
		s.p95 = durs[len(durs)*95/100]
		s.p99 = durs[len(durs)*99/100]
		sum := time.Duration(0)
		for _, d := range durs {
			sum += d
		}
		s.avg = sum / time.Duration(len(durs))
	}
	if total > 0 {
		s.throughputRPS = float64(s.ok) / total.Seconds()
	}
	return s
}

func printReport(s summary, impl string) {
	fmt.Printf("\n--- results [%s] ---\n", impl)
	fmt.Printf("total:       %d\n", s.total)
	fmt.Printf("success:     %d (%.1f%%)\n", s.ok, float64(s.ok)*100/float64(s.total))
	fmt.Printf("failure:     %d (%.1f%%)\n", s.fail, float64(s.fail)*100/float64(s.total))
	if s.ok > 0 {
		fmt.Printf("throughput:  %.2f rps\n", s.throughputRPS)
		fmt.Printf("latency:     min=%v avg=%v max=%v\n", s.min, s.avg, s.max)
		fmt.Printf("             p50=%v p95=%v p99=%v\n", s.p50, s.p95, s.p99)
	}
	if len(s.errs) > 0 {
		fmt.Printf("\n--- error breakdown ---\n")
		for msg, cnt := range s.errs {
			fmt.Printf("  %3d × %s\n", cnt, msg)
		}
	}
}
