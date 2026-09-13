package app

import (
	"context"
	"log"

	"jota/server/internal/env"
	"jota/server/internal/kv"
	"jota/server/internal/music"
	"jota/server/internal/services/spotify"
	"jota/server/internal/services/sync"
	"jota/server/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	ctx     context.Context
	Spotify *spotify.SpotifyService
	Music   *music.MusicRepository
	YouTube *youtube.Service
	Sync    *sync.Service
}

func New() *App {
	cfg := env.Load()
	spotifySvc := spotify.NewSpotifyService(cfg.SpotifyClientID)
	return &App{
		Spotify: spotifySvc,
		Music:   music.NewMusicRepository(spotifySvc),
		YouTube: youtube.NewService(),
		Sync:    sync.New(),
	}
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	kv.Start()
	if err := a.Spotify.Connect(ctx); err != nil {
		log.Printf("spotify: not connected at startup: %v", err)
	} else {
		log.Printf("spotify: connected as %s", a.Spotify.Username())
	}
	return nil
}

func (a *App) ServiceShutdown() error {
	kv.Close()
	return nil
}

// ----- Spotify bindings -----

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

func (a *App) SpotifyReconnect() error {
	return a.Spotify.Reconnect(a.ctx)
}

func (a *App) SpotifyDisconnect() error {
	return a.Spotify.Disconnect()
}

// OpenURL opens the URL in the system browser. On Android it uses the app's
// native Intent.ACTION_VIEW so OAuth runs outside the WebView; on other
// platforms the frontend opens the URL itself.
func (a *App) OpenURL(url string) error {
	return openExternal(url)
}

// ----- Music bindings -----

func (a *App) GetFullPlaylist(id string) (music.Playlist, error) {
	return a.Music.GetFullPlaylist(id)
}

func (a *App) GetFullPlaylistNoCache(id string) (music.Playlist, error) {
	return a.Music.GetFullPlaylistNoCache(id)
}

func (a *App) RevalidateFullPlaylist(id string) error {
	return a.Music.RevalidateFullPlaylist(id)
}

func (a *App) GetPlaylist(id string, page int, size int) (music.Playlist, error) {
	return a.Music.GetPlaylist(id, page, size)
}

func (a *App) GetUserPlaylists(user string) ([]music.PlaylistSummary, error) {
	return a.Music.GetUserPlaylists(user)
}

func (a *App) GetUserPlaylistsNoCache(user string) ([]music.PlaylistSummary, error) {
	return a.Music.GetUserPlaylistsNoCache(user)
}

func (a *App) RevalidateUserPlaylists(user string) error {
	return a.Music.RevalidateUserPlaylists(user)
}

func (a *App) GetSong(id string) (music.Song, error) {
	return a.Music.GetSong(id)
}

func (a *App) Search(query string, searchType string) ([]music.SearchResult, error) {
	return a.Music.Search(query, searchType)
}

func (a *App) GetArtist(uri string) (music.ArtistInfo, error) {
	return a.Music.GetArtist(uri)
}

func (a *App) GetArtistDiscography(uri string) (music.ArtistDiscography, error) {
	return a.Music.GetArtistDiscography(uri)
}

func (a *App) GetAlbumTracks(uri string) ([]music.Song, error) {
	return a.Music.GetAlbumTracks(uri)
}

// ----- YouTube bindings -----

func (a *App) GetYouTubeAudio(spotifyId string, search string) (youtube.Audio, error) {
	return a.YouTube.GetAudio(spotifyId, search)
}

func (a *App) SetYouTubeId(spotifyId string, youtubeId string) error {
	return a.YouTube.SetYoutubeId(spotifyId, youtubeId)
}

func (a *App) SearchYouTube(query string) ([]youtube.Video, error) {
	return youtube.Search(query)
}

// ----- Sync bindings -----

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

func (a *App) ReadClipboard() string {
	app := application.Get()
	if app == nil {
		return ""
	}
	text, _ := app.Clipboard.Text()
	return text
}
