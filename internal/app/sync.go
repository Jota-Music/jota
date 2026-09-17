package app

import "github.com/Jota-Music/jota/internal/services/sync"

func (a *App) SyncCheck(relayURL string) (bool, error) {
	return a.Sync.Check(relayURL)
}

func (a *App) SyncConnect(relayURL string, room string, role string, token string, password string) error {
	return a.Sync.Connect(relayURL, room, role, token, password)
}

func (a *App) SyncStop() {
	a.Sync.Stop()
}

func (a *App) SyncSend(payload string) error {
	return a.Sync.Send(payload)
}

// SyncRoomStatus reports whether a saved room is live and how many members it
// has, without joining it.
func (a *App) SyncRoomStatus(relayURL string, room string, token string) (sync.RoomStatus, error) {
	return a.Sync.RoomStatus(relayURL, room, token)
}
