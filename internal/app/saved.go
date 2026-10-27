package app

import "github.com/Jota-Music/jota/internal/music"

// Saved playlists are external YouTube or Spotify playlists the user kept in
// the library. The store is source-agnostic: the id prefix routes the lookup.

func (a *App) GetSavedPlaylists() ([]music.PlaylistSummary, error) {
	return a.Saved.List()
}

// AddSavedPlaylist keeps an external playlist in the library. The id is resolved
// to its summary by the owning source, so the store holds what the source says
// rather than what the caller claimed.
func (a *App) AddSavedPlaylist(id string) error {
	summary, err := a.Catalog.GetPlaylistSummary(id)
	if err != nil {
		return err
	}
	if err := a.Saved.Add(summary); err != nil {
		return err
	}
	a.syncInBackground()
	return nil
}

func (a *App) RemoveSavedPlaylist(id string) error {
	err := a.Saved.Remove(id)
	if err == nil {
		a.syncInBackground()
	}
	return err
}
