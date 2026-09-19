package app

import "github.com/Jota-Music/jota/internal/rooms"

func (a *App) ListRooms() ([]rooms.Room, error) {
	return a.Rooms.List()
}

func (a *App) SaveRoom(room rooms.Room) ([]rooms.Room, error) {
	return a.Rooms.Save(room)
}

func (a *App) RemoveRoom(id string) ([]rooms.Room, error) {
	return a.Rooms.Remove(id)
}
