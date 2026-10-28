package app

import (
	"github.com/Jota-Music/jota/internal/music"
)

func (a *App) GetPlaylists() ([]music.PlaylistSummary, error) {
	return a.Playlists.List()
}

func (a *App) CreatePlaylist(name string) (music.PlaylistSummary, error) {
	summary, err := a.Playlists.Create(name)
	if err == nil {
		a.syncInBackground()
	}
	return summary, err
}

func (a *App) RenamePlaylist(id, name string) error {
	err := a.Playlists.Rename(id, name)
	if err == nil {
		a.syncInBackground()
	}
	return err
}

func (a *App) DeletePlaylist(id string) error {
	err := a.Playlists.Delete(id)
	if err == nil {
		a.syncInBackground()
	}
	return err
}

func (a *App) AddSongsToPlaylist(id string, refs []string) error {
	err := a.Playlists.AddSongs(id, refs)
	if err == nil {
		a.syncInBackground()
	}
	return err
}

func (a *App) RemoveSongsFromPlaylist(id string, refs []string) error {
	err := a.Playlists.RemoveSongs(id, refs)
	if err == nil {
		a.syncInBackground()
	}
	return err
}

func (a *App) ReorderPlaylist(id string, refs []string) error {
	err := a.Playlists.Reorder(id, refs)
	if err == nil {
		a.syncInBackground()
	}
	return err
}
