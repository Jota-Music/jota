package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// spotifyService is the package-level handle used by sync apply, which has no
// App instance. It is set during New().
var spotifyService *spotify.SpotifyService

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
	if err := a.Spotify.CompleteLogin(); err != nil {
		return err
	}
	a.syncSpotifySession()
	return nil
}

func (a *App) SpotifyDisconnect() error {
	err := a.Spotify.Disconnect()
	tombstoneSession("spotify")
	a.syncInBackground()
	return err
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

func (a *App) syncSpotifySession() {
	pass := a.SessionPass()
	if pass == "" {
		return
	}
	username, data, err := a.Spotify.StoredSession()
	if err != nil || username == "" {
		return
	}
	payload, err := json.Marshal(map[string]string{
		"username": username,
		"data":     hex.EncodeToString(data),
	})
	if err != nil {
		return
	}
	// Same gate as the YouTube cookies: an identical plaintext proves the
	// staged session already reflects this login, so there is nothing to stage.
	if stored := stagedSession("spotify"); stored != "" {
		if pt, err := decryptWithPass(pass, stored); err == nil && pt == string(payload) {
			return
		}
	}
	enc, err := encryptWithPass(pass, string(payload))
	if err != nil {
		return
	}
	if err := settings.SetObject("session:spotify", enc); err != nil {
		return
	}
	_ = settings.SetObject("session:spotify"+settingAt, time.Now().UnixMilli())
	sessionsSync.dirty.Store(true)
	a.syncInBackground()
}

func applySpotifySession(blob string) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return
	}
	pass := storedPass()
	if pass == "" {
		return
	}
	pt, err := decryptWithPass(pass, blob)
	if err != nil {
		return
	}
	var p struct {
		Username string `json:"username"`
		Data     string `json:"data"`
	}
	if err := json.Unmarshal([]byte(pt), &p); err != nil {
		return
	}
	if p.Username == "" || p.Data == "" {
		return
	}
	data, err := hex.DecodeString(p.Data)
	if err != nil {
		return
	}
	if spotifyService == nil {
		return
	}
	// Only touch the session when the incoming copy differs from what this
	// device already holds, so a pass that carries nothing new does not
	// disconnect a working session.
	if curUser, curData, err := spotifyService.StoredSession(); err == nil {
		if curUser == p.Username && bytes.Equal(curData, data) {
			return
		}
	}
	go func() {
		if !spotifyService.AdoptSession(p.Username, data) {
			return
		}
		if app := application.Get(); app != nil {
			app.Event.Emit("spotify:reconnected")
		}
	}()
}
