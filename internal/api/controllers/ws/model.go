package ws_controller

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/contrib/v3/websocket"
)

var connMu sync.Map

// type Message[T any] struct {
// 	From   From   `json:"from"`
// 	Action string `json:"action"`
// 	Params T      `json:"params"`
// }

type Response[T any] struct {
	From   From   `json:"from"`
	Action string `json:"action"`
	Data   T      `json:"data"`
}

type Message struct {
	From   From            `json:"from"`
	Action string          `json:"action"`
	Params json.RawMessage `json:"params"`
}

type JoinedSessionData struct {
	YouAreOwner      bool   `json:"youAreOwner"`
	AwaitingSnapshot bool   `json:"awaitingSnapshot"`
	Visibility       string `json:"visibility"` // "public" | "private"
	Headcount        int    `json:"headcount"`
}

type RoomCountPayload struct {
	Count int `json:"count"`
}

type SocketErrorResponse struct {
	Status  string `json:"status" default:"error"`
	Success bool   `json:"success"`
	Error   string `json:"errors"`
	Action  string `json:"action"`
}

type From string

const (
	Owner      From = "owner"
	Subscriber From = "subscriber"
)

func Send[T any](c *websocket.Conn, from From, action string, data T) {
	mu, _ := connMu.LoadOrStore(c, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	c.WriteJSON(Response[T]{
		From:   from,
		Action: action,
		Data:   data,
	})
}

func SendError(c *websocket.Conn, action string, e string) {
	mu, _ := connMu.LoadOrStore(c, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	c.WriteJSON(SocketErrorResponse{
		Success: false,
		Error:   e,
		Action:  action,
	})
}
