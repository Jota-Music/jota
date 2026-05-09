package ws_controller

import (
	"github.com/gofiber/contrib/v3/websocket"
)

type SnapshotData struct {
	Queue    []any    `json:"queue"`
	Index    int      `json:"index"`
	Playing  bool     `json:"playing"`
	Position *float64 `json:"position,omitempty"`
}

func Toggle(c *websocket.Conn, roomId string) {
	Broadcast(c, roomId, "toggle", nil)
}

func Snapshot(c *websocket.Conn, roomId string, sn SnapshotData) {
	Broadcast(c, roomId, "snapshot", sn)
}

func Seek(c *websocket.Conn, roomId string, position float64) {
	Broadcast(c, roomId, "seek", map[string]float64{"position": position})
}
