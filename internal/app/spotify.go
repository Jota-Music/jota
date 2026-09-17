package app

import "github.com/Jota-Music/jota/internal/music"

type SpotifyStatus struct {
	Connected bool   `json:"connected"`
	User      string `json:"user"`
}

func (a *App) SpotifyGetStatus() SpotifyStatus {
	return SpotifyStatus{
		Connected: a.Spotify.IsConnected(),
		User:      a.Spotify.Username(),
	}
}

// SpotifyLogin starts the OAuth flow. It launches a local callback server,
// generates the auth URL pointing to it, and returns that URL. The frontend
// should open it in the system browser. When the user completes the flow
// in their browser, Spotify redirects to the local callback server which
// captures the code and completes the login. SpotifyLoginAndWait blocks
// until the flow finishes (success or timeout).
func (a *App) SpotifyLogin() (string, error) {
	return a.Spotify.StartupLogin()
}

func (a *App) SpotifyLoginAndWait() error {
	return a.Spotify.CompleteLogin()
}

func (a *App) SpotifyDisconnect() error {
	return a.Spotify.Disconnect()
}

func (a *App) GetUserPlaylists(user string) ([]music.PlaylistSummary, error) {
	return a.Spotify.GetUserPlaylists(user)
}

func (a *App) RevalidateUserPlaylists(user string) error {
	return a.Spotify.RevalidateUserPlaylists(user)
}

func (a *App) GetUserProfile(username string) (music.UserProfile, error) {
	return a.Spotify.GetUserProfile(username)
}

func (a *App) GetFriends() ([]string, error) {
	return a.Spotify.GetFriends()
}

func (a *App) GetFollowing() ([]music.Follow, error) {
	return a.Spotify.GetFollowing()
}

func (a *App) Search(query string, searchType string) ([]music.SearchResult, error) {
	return a.Spotify.Search(query, searchType)
}

func (a *App) GetArtist(uri string) (music.ArtistInfo, error) {
	return a.Spotify.GetArtist(uri)
}

func (a *App) GetArtistDiscography(uri string) (music.ArtistDiscography, error) {
	return a.Spotify.GetArtistDiscography(uri)
}

func (a *App) GetAlbumTracks(uri string) ([]music.Song, error) {
	return a.Spotify.GetAlbumTracks(uri)
}
