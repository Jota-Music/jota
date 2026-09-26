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
	return a.Saved.Add(summary)
}

func (a *App) RemoveSavedPlaylist(id string) error {
	return a.Saved.Remove(id)
}
