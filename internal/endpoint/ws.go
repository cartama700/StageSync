package endpoint

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/kimsehoon/stagesync/api/proto/roompb"
	"github.com/kimsehoon/stagesync/internal/room"
)

// WSHandler — HTTP 요청을 WebSocket 으로 업그레이드하고
// protobuf ClientMessage 프레임을 Room 으로 분배.
type WSHandler struct {
	Room *room.Room
}

// ServeHTTP — chi 의 GET("/ws/room", ...) 에 등록.
// 연결 후 끊길 때까지 Read 루프로 ClientMessage 수신.
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// 개발용. 프로덕션은 구체 오리진으로 제한.
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		slog.Error("websocket accept failed", "err", err)
		return
	}
	defer conn.CloseNow()

	ctx := r.Context()
	playerID := ""
	slog.Info("websocket connected", "remote", r.RemoteAddr)

	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) ||
				websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				slog.Info("websocket closed", "player", playerID)
			} else {
				slog.Info("websocket read ended", "err", err, "player", playerID)
			}
			break
		}
		if msgType != websocket.MessageBinary {
			slog.Warn("ignoring non-binary frame", "type", msgType.String())
			continue
		}

		var msg roompb.ClientMessage
		if err := proto.Unmarshal(data, &msg); err != nil {
			slog.Warn("proto unmarshal failed", "err", err)
			continue
		}

		switch p := msg.Payload.(type) {
		case *roompb.ClientMessage_Move:
			playerID = p.Move.GetPlayerId()
			h.Room.ApplyMove(p.Move)
			slog.Info("move received",
				"player", playerID,
				"x", p.Move.GetX(),
				"y", p.Move.GetY(),
				"room_size", h.Room.Size(),
			)
		default:
			slog.Warn("unknown payload type")
		}
	}

	if playerID != "" {
		h.Room.Remove(playerID)
		slog.Info("player removed", "player", playerID, "room_size", h.Room.Size())
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		slog.Debug("websocket close err", "err", err)
	}
}
