package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	stdsync "sync"
	"time"

	"github.com/Jota-Music/jota/internal/env"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/services/follows"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/Jota-Music/jota/internal/services/sync"
	"github.com/Jota-Music/jota/internal/services/update"
	"github.com/Jota-Music/jota/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type App struct {
	ctx     context.Context
	version string
	Spotify *spotify.SpotifyService
	Catalog *music.Catalog
	YouTube *youtube.Service
	Sync    *sync.Service
	Follows *follows.Service
}

var installMu stdsync.Mutex

func New(version string) *App {
	cfg := env.Load()
	spotifySvc := spotify.NewSpotifyService(cfg.SpotifyClientID)
	youTubeSvc := youtube.NewService()
	return &App{
		version: version,
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

// ----- App bindings -----

func (a *App) Version() string {
	return a.version
}

func (a *App) CheckUpdate() (update.Info, error) {
	return update.Check(a.version)
}

// InstallUpdate downloads and swaps in the newest release, then quits so the
// relaunch scheduled by the updater can start the new version.
func (a *App) InstallUpdate() error {
	if !installMu.TryLock() {
		return errors.New("update: install already in progress")
	}
	defer installMu.Unlock()

	if err := update.Install(a.version, func(p update.Progress) {
		application.Get().Event.Emit("update:progress", p)
	}); err != nil {
		return err
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		application.Get().Quit()
	}()
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

		// A sign-in would fix it, so nudge the UI to offer one.
		if errors.Is(err, youtube.ErrLoginRequired) && !youtube.HasAuth() {
			application.Get().Event.Emit("youtube:signin-required", song.Name)
		}
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

// SetYouTubeCookies stores the Google account cookies used to sign innertube
// requests. Playing age-restricted videos and avoiding the bot check requires
// it; using an account this way can get it banned by Google.
func (a *App) SetYouTubeCookies(cookies string) error {
	return youtube.SetCookies(cookies)
}

func (a *App) ClearYouTubeCookies() error {
	return youtube.ClearCookies()
}

// YouTubeBrowserLogin opens a window on youtube.com so the user can sign in
// normally; the page beacons its document.cookie back to a local listener, which
// is stored for innertube requests. Blocks until signed in, the window is
// closed, or it times out.
func (a *App) YouTubeBrowserLogin() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("login listener: %w", err)
	}
	defer ln.Close()

	cookies := make(chan string, 8)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			_ = r.ParseForm()
			value := r.FormValue("c")
			if value == "" {
				body, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
				value = string(body)
			}
			log.Printf("youtube: browser login received %d bytes", len(value))
			if value != "" {
				select {
				case cookies <- value:
				default:
				}
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<!doctype html><title>Jota</title><p>Signed in. You can close this window.</p>")
		}),
	}
	defer server.Close()
	go func() { _ = server.Serve(ln) }()

	// ExecJS is gated on the page loading the Wails runtime, which an external
	// page never does, so inject the collector through WebviewWindowOptions.JS:
	// it runs on every load. Once SAPISID shows up it hands document.cookie over
	// as a top-level form POST, which sidesteps CORS, PNA and mixed content.
	collector := fmt.Sprintf(`(function () {
  var target = %q;
  function send() {
    try {
      if (document.cookie.indexOf("SAPISID=") === -1 &&
          document.cookie.indexOf("__Secure-3PAPISID=") === -1) return;
      var form = document.createElement("form");
      form.method = "POST";
      form.action = target;
      var field = document.createElement("textarea");
      field.name = "c";
      field.value = document.cookie;
      form.appendChild(field);
      (document.body || document.documentElement).appendChild(form);
      form.submit();
    } catch (e) {}
  }
  setInterval(send, 1500);
  send();
})();`, fmt.Sprintf("http://127.0.0.1:%d/", ln.Addr().(*net.TCPAddr).Port))

	window := application.Get().Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Sign in to YouTube",
		URL:    "https://www.youtube.com/",
		Width:  520,
		Height: 760,
		JS:     collector,
	})
	defer window.Close()

	closed := make(chan struct{})
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		select {
		case <-closed:
		default:
			close(closed)
		}
	})

	timeout := time.After(3 * time.Minute)
	for {
		select {
		case raw := <-cookies:
			if err := youtube.SetCookies(raw); err != nil {
				log.Printf("youtube: browser login cookies rejected: %v", err)
				continue
			}
			return nil
		case <-closed:
			return errors.New("login window closed")
		case <-timeout:
			return errors.New("login timed out")
		}
	}
}

func (a *App) YouTubeSignedIn() bool {
	return youtube.HasAuth()
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
