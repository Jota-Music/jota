package handler

import (
	"encoding/json"
	"errors"
	ws_controller "jota/server/internal/api/controllers/ws"
	"jota/server/internal/music"
	"jota/server/internal/session"
	"jota/server/internal/utils/ws"
	"log"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

type snapshotParams struct {
	Queue    []music.Song `json:"queue"`
	Index    int          `json:"index"`
	Playing  bool         `json:"playing"`
	Position *float64     `json:"position,omitempty"`
}

func WS(app *fiber.App) {
	app.Get("/ws/:room", websocket.New(func(c *websocket.Conn) {
		room := c.Params("room")
		if room == "" {
			c.WriteJSON(ws_controller.Response[string]{
				Action: "error",
				Data:   "room is required",
			})
			return
		}

		if err := ws_controller.ValidateRoomAccess(room, c.Cookies(session.CookieName)); err != nil {
			if errors.Is(err, ws_controller.ErrRoomForbidden) {
				ws_controller.Send(c, ws_controller.Subscriber, "room-forbidden",
					"this room name belongs to another user")
			} else {
				ws_controller.Send(c, ws_controller.Subscriber, "error", err.Error())
			}
			_ = c.Close()
			return
		}

		defer func() {
			log.Printf("ws: disconnected (room %s)", room)
			ws_controller.Leave(c, room)
			c.Close()
		}()

		for {
			msg, err := ws.ParseJSON[ws_controller.Message](c)
			if err != nil {
				c.WriteJSON(ws_controller.Response[string]{
					Action: "error",
					Data:   err.Error(),
				})
				break
			}

			switch msg.Action {
			case "join":
				log.Printf("ws: connected (room %s)", room)
				ws_controller.Join(c, room)
			case "leave":
				log.Printf("ws: executing leave (room %s)", room)
				ws_controller.Leave(c, room)
			case "toggle":
				log.Printf("ws: executing toggle (room %s)", room)
				ws_controller.Toggle(c, room)
			case "snapshot":
				log.Printf("ws: snapshot (room %s)", room)
				var p snapshotParams
				if err := json.Unmarshal(msg.Params, &p); err != nil {
					log.Printf("ws: snapshot bad params (room %s): %v", room, err)
					continue
				}
				sn := ws_controller.SnapshotData{
					Queue:    make([]any, len(p.Queue)),
					Index:    p.Index,
					Playing:  p.Playing,
					Position: p.Position,
				}
				for i := range p.Queue {
					sn.Queue[i] = p.Queue[i]
				}
				ws_controller.Snapshot(c, room, sn)
			case "seek":
				log.Printf("ws: seek (room %s)", room)
				var p struct {
					Position float64 `json:"position"`
				}
				if err := json.Unmarshal(msg.Params, &p); err != nil {
					log.Printf("ws: seek bad params (room %s): %v", room, err)
					continue
				}
				ws_controller.Seek(c, room, p.Position)
			case "new-track":
				log.Printf("ws: new-track (room %s)", room)
				var p struct {
					Queue    []music.Song `json:"queue"`
					Index    int          `json:"index"`
					Playing  bool         `json:"playing"`
					Position *float64     `json:"position,omitempty"`
					TrackId  string       `json:"trackId"`
				}
				if err := json.Unmarshal(msg.Params, &p); err != nil {
					log.Printf("ws: new-track bad params (room %s): %v", room, err)
					continue
				}
				sn := ws_controller.SnapshotData{
					Queue:    make([]any, len(p.Queue)),
					Index:    p.Index,
					Playing:  p.Playing,
					Position: p.Position,
				}
				for i := range p.Queue {
					sn.Queue[i] = p.Queue[i]
				}
				ws_controller.Broadcast(c, room, "new-track", sn)
				ws_controller.TrackNewSong(c, room, p.TrackId)
			case "ready":
				log.Printf("ws: ready (room %s)", room)
				ws_controller.ConnReady(c, room)
			case "set-visibility":
				var p struct {
					Visibility string `json:"visibility"`
				}
				if err := json.Unmarshal(msg.Params, &p); err != nil {
					log.Printf("ws: set-visibility bad params (room %s): %v", room, err)
					continue
				}
				if p.Visibility != "public" && p.Visibility != "private" {
					log.Printf("ws: set-visibility invalid visibility %q (room %s)", p.Visibility, room)
					continue
				}
				ws_controller.SetRoomVisibility(c, room, p.Visibility == "private", c.Cookies(session.CookieName))
			default:
				c.WriteJSON(ws_controller.Response[string]{
					Action: "unknown",
					Data:   "action not found",
				})
			}
		}
	}))
}
