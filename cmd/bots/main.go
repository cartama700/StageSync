// cmd/bots — WebSocket 부하 시뮬 봇.
// ws://host/ws/room 에 N 개 봇을 연결해서 시나리오 별 Move 패턴 송신.
//
// 시나리오:
//
//	even     — 봇이 맵 전체에 균등 분산 이동 (기본값)
//	herd     — 모든 봇이 동일 좌표 주변으로 뭉쳐서 이동 (핫스팟 부하 테스트)
//	cluster  — 봇 그룹이 여러 중심점 주변으로 군집 (중간 부하)
//
// 예:
//
//	go run ./cmd/bots -n=50 -scenario=herd -tick=100
//	go run ./cmd/bots -n=1 (단일 봇 — Phase 1 동작 확인)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/kimsehoon/stagesync/api/proto/roompb"
)

func main() {
	if err := run(); err != nil {
		slog.Error("bots fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr     = flag.String("addr", "ws://localhost:5050/ws/room", "server WebSocket address")
		nBots    = flag.Int("n", 1, "number of bots")
		tickMs   = flag.Int("tick", 200, "tick interval in milliseconds")
		scenario = flag.String("scenario", "even", "scenario: even | herd | cluster")
		seed     = flag.Int64("seed", 0, "rng seed (0 = time-based)")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if *nBots < 1 {
		return fmt.Errorf("-n must be >= 1, got %d", *nBots)
	}
	scen, err := parseScenario(*scenario)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rngSeed := *seed
	if rngSeed == 0 {
		rngSeed = time.Now().UnixNano()
	}

	slog.Info("bots starting",
		"addr", *addr, "n", *nBots, "scenario", *scenario, "tick_ms", *tickMs, "seed", rngSeed,
	)

	var wg sync.WaitGroup
	wg.Add(*nBots)
	for i := 0; i < *nBots; i++ {
		i := i
		go func() {
			defer wg.Done()
			playerID := "p" + strconv.Itoa(i)
			// #nosec G404 — 부하 시뮬 RNG, 암호학적 보안 불필요.
			botRng := rand.New(rand.NewPCG(uint64(rngSeed), uint64(i+1)))
			if err := runBot(ctx, *addr, playerID, time.Duration(*tickMs)*time.Millisecond, scen, botRng); err != nil {
				slog.Error("bot exited with error", "player", playerID, "err", err)
			}
		}()
	}
	wg.Wait()
	slog.Info("all bots stopped")
	return nil
}

// scenario — 각 봇의 x, y 좌표 계산 전략.
type scenario func(rng *rand.Rand, tick uint64) (x, y float64)

func parseScenario(s string) (scenario, error) {
	switch s {
	case "even":
		return evenScenario, nil
	case "herd":
		return herdScenario, nil
	case "cluster":
		return clusterScenario, nil
	default:
		return nil, fmt.Errorf("unknown scenario %q (want even|herd|cluster)", s)
	}
}

// evenScenario — 맵 전체 [-500, 500]² 를 균등히 돌아다님.
func evenScenario(rng *rand.Rand, _ uint64) (float64, float64) {
	return rng.Float64()*1000 - 500, rng.Float64()*1000 - 500
}

// herdScenario — 원점 근처 [-20, 20]² 로 몰림 (핫스팟).
func herdScenario(rng *rand.Rand, _ uint64) (float64, float64) {
	return rng.Float64()*40 - 20, rng.Float64()*40 - 20
}

// clusterScenario — 5 개 중심점 중 하나로 군집, 각 중심에서 [-30, 30]² 흔들림.
func clusterScenario(rng *rand.Rand, _ uint64) (float64, float64) {
	centers := [5][2]float64{
		{-300, -300}, {300, -300}, {0, 0}, {-300, 300}, {300, 300},
	}
	c := centers[rng.IntN(len(centers))]
	return c[0] + math.Round(rng.Float64()*60-30), c[1] + math.Round(rng.Float64()*60-30)
}

// runBot — 한 봇의 라이프사이클. ctx 취소 시 정상 종료.
func runBot(ctx context.Context, addr, playerID string, tick time.Duration, scen scenario, rng *rand.Rand) error {
	conn, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer func() { _ = conn.CloseNow() }()

	slog.Info("bot connected", "player", playerID)

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	var step uint64
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "bye")
			return nil
		case <-ticker.C:
			step++
			x, y := scen(rng, step)
			msg := &roompb.ClientMessage{
				Payload: &roompb.ClientMessage_Move{
					Move: &roompb.Move{PlayerId: playerID, X: x, Y: y},
				},
			}
			data, err := proto.Marshal(msg)
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}
			if err := conn.Write(ctx, websocket.MessageBinary, data); err != nil {
				return fmt.Errorf("write: %w", err)
			}
		}
	}
}
