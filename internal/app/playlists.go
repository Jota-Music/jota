package app

import "github.com/Jota-Music/jota/internal/music"

func (a *App) GetPlaylists() ([]music.PlaylistSummary, error) {
	return a.Playlists.List()
}

func (a *App) CreatePlaylist(name string) (music.PlaylistSummary, error) {
	return a.Playlists.Create(name)
}

func (a *App) DeletePlaylist(id string) error {
	return a.Playlists.Delete(id)
}

func (a *App) AddSongsToPlaylist(id string, refs []string) error {
	return a.Playlists.AddSongs(id, refs)
}

func (a *App) RemoveSongFromPlaylist(id string, ref string) error {
	return a.Playlists.RemoveSong(id, ref)
}

func (a *App) ReorderPlaylist(id string, refs []string) error {
	return a.Playlists.Reorder(id, refs)
}
