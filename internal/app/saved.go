package app

import "github.com/Jota-Music/jota/internal/music"

// Saved playlists are external YouTube or Spotify playlists the user kept in
// the library. The store is source-agnostic: the id prefix routes the lookup.

func (a *App) GetSavedPlaylists() ([]music.PlaylistSummary, error) {
	return a.Saved.List()
}

func (a *App) RemoveSavedPlaylist(id string) error {
	return a.Saved.Remove(id)
}
