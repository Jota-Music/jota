package ws_controller

import (
	"errors"
	"jota/server/internal/kv"
	"jota/server/internal/repositories"
	"jota/server/internal/session"
)

// ErrRoomForbidden is returned when the room id matches a registered user
// but the client may not connect (wrong account, private room, or room not open yet).
var ErrRoomForbidden = errors.New("room name reserved for another user")

// ValidateRoomAccess: arbitrary ids (e.g. UUID) are allowed. A registered username as
// room id is allowed for that logged-in user, or for anyone if the room already exists
// and is public (guests can join). Otherwise forbidden.
func ValidateRoomAccess(roomName, sessionCookie string) error {
	_, err := repositories.Use.Auth.GetUser(roomName)
	if err != nil {
		if errors.Is(err, kv.KeyNotFoundError) {
			return nil
		}
		return err
	}
	sessionName, sessErr := session.ValidateAndTouch(sessionCookie)
	if sessErr == nil && sessionName == roomName {
		return nil
	}
	if r := rooms[roomName]; r != nil && !r.Private {
		return nil
	}
	return ErrRoomForbidden
}
