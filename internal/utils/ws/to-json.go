package ws

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/contrib/v3/websocket"
)

func ParseJSON[T any](c *websocket.Conn) (T, error) {
	var result T

	// Read raw message first
	_, data, err := c.ReadMessage()
	if err != nil {
		return result, err
	}

	// json.Unmarshal ignores unknown JSON keys for structs (unlike json.Decoder
	// with DisallowUnknownFields, which rejects extra root fields).
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("invalid payload for %T: %w", result, err)
	}

	return result, nil
}
