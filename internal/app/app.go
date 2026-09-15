package app

import (
	"context"
	"log"

	"github.com/Jota-Music/jota/internal/env"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/services/follows"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/Jota-Music/jota/internal/services/sync"
	"github.com/Jota-Music/jota/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	ctx     context.Context
	Spotify *spotify.SpotifyService
	Catalog *music.Catalog
	YouTube *youtube.Service
	Sync    *sync.Service
	Follows *follows.Service
}

func New() *App {
	cfg := env.Load()
	spotifySvc := spotify.NewSpotifyService(cfg.SpotifyClientID)
	youTubeSvc := youtube.NewService()
	return &App{
		Spotify: spotifySvc,
		Catalog: music.NewCatalog(spotifySvc, youTubeSvc),
		YouTube: youTubeSvc,
		Sync:    sync.New(),
		Follows: follows.New(),
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
	return a.Catalog.GetFullPlaylist(id)
}

func (a *App) GetFullPlaylistNoCache(id string) (music.Playlist, error) {
	return a.Catalog.GetFullPlaylistNoCache(id)
}

func (a *App) RevalidateFullPlaylist(id string) error {
	return a.Catalog.RevalidateFullPlaylist(id)
}

func (a *App) GetUserPlaylists(user string) ([]music.PlaylistSummary, error) {
	return a.Spotify.GetUserPlaylists(user)
}

func (a *App) GetUserPlaylistsNoCache(user string) ([]music.PlaylistSummary, error) {
	return a.Spotify.GetUserPlaylistsNoCache(user)
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

// ----- Follows bindings -----

func (a *App) GetFollowedUsers(account string) ([]string, error) {
	return a.Follows.List(account)
}

func (a *App) FollowUser(account string, user string) error {
	return a.Follows.Follow(account, user)
}

func (a *App) UnfollowUser(account string, user string) error {
	return a.Follows.Unfollow(account, user)
}

func (a *App) GetSong(id string) (music.Song, error) {
	return a.Spotify.GetSong(id)
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

// ----- YouTube bindings -----

func (a *App) ResolveAudio(song music.Song) (music.Audio, error) {
	audio, err := a.YouTube.ResolveAudio(song)
	if err != nil {
		log.Printf("resolve-audio failed id=%q youtubeId=%q name=%q err=%v",
			song.Id, song.YoutubeId, song.Name, err)
	}
	return audio, err
}

func (a *App) SetYouTubeId(cacheKey string, youtubeId string) error {
	return a.YouTube.SetYoutubeId(cacheKey, youtubeId)
}

func (a *App) SearchYouTube(query string) ([]youtube.Video, error) {
	return a.YouTube.Search(query)
}

func (a *App) SearchYouTubePlaylists(query string) ([]music.PlaylistSummary, error) {
	return a.YouTube.SearchPlaylists(query)
}

func (a *App) GetYouTubePlaylists() ([]music.PlaylistSummary, error) {
	return a.YouTube.Playlists()
}

func (a *App) AddYouTubePlaylist(id string) (music.PlaylistSummary, error) {
	return a.YouTube.AddPlaylist(id)
}

func (a *App) RemoveYouTubePlaylist(id string) error {
	return a.YouTube.RemovePlaylist(id)
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
