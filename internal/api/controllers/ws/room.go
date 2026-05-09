package ws_controller

import (
	"errors"

	"github.com/gofiber/contrib/v3/websocket"

	"jota/server/internal/kv"
	"jota/server/internal/repositories"
	"jota/server/internal/session"
)

type Room struct {
	Id      string
	Owner   *websocket.Conn
	Guests  []*websocket.Conn
	Private bool
}

var rooms = make(map[string]*Room)

func NewRoom(id string, owner *websocket.Conn) *Room {
	newRoom := &Room{
		Id:      id,
		Owner:   owner,
		Guests:  []*websocket.Conn{},
		Private: false,
	}

	rooms[id] = newRoom
	return newRoom
}

func roomVisibility(r *Room) string {
	if r.Private {
		return "private"
	}
	return "public"
}

func notifyRoomCount(roomId string) {
	room := rooms[roomId]
	if room == nil {
		return
	}
	notifyRoom(roomId, "room-count", RoomCountPayload{Count: len(room.Guests)})
}

func Join(c *websocket.Conn, roomId string) {
	room := rooms[roomId]
	if room == nil {
		room = NewRoom(roomId, c)
		room.Guests = append(room.Guests, c)
		Send(c, Owner, "joined", JoinedSessionData{
			YouAreOwner:      true,
			AwaitingSnapshot: false,
			Visibility:       roomVisibility(room),
			Headcount:        len(room.Guests),
		})
		return
	}

	if room.Private && c != room.Owner {
		Send(c, Owner, "join-denied", "room is private")
		_ = c.Close()
		return
	}

	room.Guests = append(room.Guests, c)

	youAreOwner := room.Owner == c
	awaitingSnapshot := !youAreOwner && room.Owner != nil

	Send(c, Owner, "joined", JoinedSessionData{
		YouAreOwner:      youAreOwner,
		AwaitingSnapshot: awaitingSnapshot,
		Visibility:       roomVisibility(room),
		Headcount:        len(room.Guests),
	})

	if room.Owner != nil && c != room.Owner {
		Send(room.Owner, Subscriber, "request-snapshot", struct{}{})
	}
	notifyRoomCount(roomId)
}

func Leave(c *websocket.Conn, roomId string) {
	room := rooms[roomId]
	if room == nil {
		return
	}

	wasOwner := room.Owner != nil && room.Owner == c

	for i, guest := range room.Guests {
		if guest == c {
			room.Guests = append(room.Guests[:i], room.Guests[i+1:]...)
			break
		}
	}

	if !wasOwner {
		if r := rooms[roomId]; r != nil {
			notifyRoomCount(roomId)
		}
		return
	}

	if len(room.Guests) == 0 {
		delete(rooms, roomId)
		return
	}

	room.Owner = room.Guests[0]
	Send(room.Owner, Subscriber, "request-snapshot", struct{}{})
	notifyRoomCount(roomId)
}

func Broadcast(c *websocket.Conn, roomId string, action string, data any) {
	room := rooms[roomId]
	if room == nil {
		return
	}

	for _, guest := range room.Guests {
		if guest == c {
			continue
		}
		Send(guest, Subscriber, action, data)
	}
}

func notifyRoom(roomId string, action string, data any) {
	room := rooms[roomId]
	if room == nil {
		return
	}
	for _, g := range room.Guests {
		Send(g, Subscriber, action, data)
	}
}

type visibilityPayload struct {
	Visibility string `json:"visibility"`
}

func canControlVisibility(c *websocket.Conn, roomId string, room *Room, sessionCookie string) bool {
	if room.Owner == c {
		return true
	}
	_, err := repositories.Use.Auth.GetUser(roomId)
	if err != nil {
		if errors.Is(err, kv.KeyNotFoundError) {
			return false
		}
		return false
	}
	sessionName, sessErr := session.ValidateAndTouch(sessionCookie)
	return sessErr == nil && sessionName == roomId
}

func SetRoomVisibility(c *websocket.Conn, roomId string, private bool, sessionCookie string) {
	room := rooms[roomId]
	if room == nil {
		SendError(c, "visibility-error", "room not found")
		return
	}
	if !canControlVisibility(c, roomId, room, sessionCookie) {
		SendError(c, "visibility-error", "only owner can change visibility")
		return
	}

	room.Private = private
	vis := roomVisibility(room)
	notifyRoom(roomId, "visibility", visibilityPayload{Visibility: vis})

	if !private {
		return
	}

	var toKick []*websocket.Conn
	for _, g := range room.Guests {
		if g != room.Owner {
			toKick = append(toKick, g)
		}
	}
	for _, g := range toKick {
		Send(g, Subscriber, "kicked", map[string]string{"reason": "room is now private"})
		_ = g.Close()
	}
	if r := rooms[roomId]; r != nil {
		notifyRoomCount(roomId)
	}
}
