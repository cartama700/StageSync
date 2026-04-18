// cmd/bots — WebSocket 부하 시뮬 봇.
// ws://host/ws/room 에 접속해서 주기적으로 Move 메시지 송신.
// Phase 1 은 봇 1개. Phase 12 에서 even/herd/cluster 시나리오로 확장 예정.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/kimsehoon/stagesync/api/proto/roompb"
)

func main() {
	var (
		addr     = flag.String("addr", "ws://localhost:5050/ws/room", "server WebSocket address")
		playerID = flag.String("player", "p1", "player id")
		tickMs   = flag.Int("tick", 200, "tick interval in milliseconds")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Ctrl+C / SIGTERM 수신 시 ctx 취소 → select 루프 탈출.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, *addr, nil)
	if err != nil {
		slog.Error("dial failed", "addr", *addr, "err", err)
		os.Exit(1)
	}
	defer conn.CloseNow()

	slog.Info("bot connected", "addr", *addr, "player", *playerID, "tick_ms", *tickMs)

	ticker := time.NewTicker(time.Duration(*tickMs) * time.Millisecond)
	defer ticker.Stop()

	x, y := 0.0, 0.0
	for {
		select {
		case <-ctx.Done():
			slog.Info("bot shutting down")
			if err := conn.Close(websocket.StatusNormalClosure, "bye"); err != nil {
				slog.Debug("close err", "err", err)
			}
			return
		case <-ticker.C:
			x += 1.0
			y += 0.5
			msg := &roompb.ClientMessage{
				Payload: &roompb.ClientMessage_Move{
					Move: &roompb.Move{
						PlayerId: *playerID,
						X:        x,
						Y:        y,
					},
				},
			}
			data, err := proto.Marshal(msg)
			if err != nil {
				slog.Error("marshal failed", "err", err)
				continue
			}
			if err := conn.Write(ctx, websocket.MessageBinary, data); err != nil {
				slog.Error("write failed", "err", err)
				return
			}
			slog.Info("move sent", "player", *playerID, "x", x, "y", y)
		}
	}
}
