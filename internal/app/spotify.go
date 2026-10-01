package app

import "github.com/Jota-Music/jota/internal/music"

type SpotifyStatus struct {
	Connected bool   `json:"connected"`
	Degraded  bool   `json:"degraded"`
	User      string `json:"user"`
}

func (a *App) SpotifyGetStatus() SpotifyStatus {
	return SpotifyStatus{
		Connected: a.Spotify.IsConnected(),
		Degraded:  a.Spotify.IsDegraded(),
		User:      a.Spotify.Username(),
	}
}

// SpotifyLogin starts the OAuth flow. It launches a local callback server,
// generates the auth URL pointing to it, and returns that URL. The frontend
// should open it in the system browser. When the user completes the flow
// in their browser, Spotify redirects to the local callback server which
// captures the code and completes the login. SpotifyLoginAndWait blocks
// until the flow finishes (success or timeout).
//
// hold() runs across the whole flow: the redirect target is a loopback
// listener inside this process, so on Android the app must survive being
// backgrounded for the browser to be able to hand the code back.
func (a *App) SpotifyLogin() (string, error) {
	url, err := a.Spotify.StartupLogin()
	if err != nil {
		return "", err
	}
	hold()
	return url, nil
}

func (a *App) SpotifyLoginAndWait() error {
	defer release()
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

func (a *App) RevalidateFollowing() {
	a.Spotify.RevalidateFollowing()
}

// SpotifyReconnect retries the session restore on demand, for the degraded
// banner's retry button. It reports whether the session is back.
func (a *App) SpotifyReconnect() bool {
	return a.Spotify.Reconnect()
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
