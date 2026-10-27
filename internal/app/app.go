package app

import (
	"context"
	"errors"
	"log"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/Jota-Music/jota/internal/env"
	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/refresh"
	"github.com/Jota-Music/jota/internal/rooms"
	"github.com/Jota-Music/jota/internal/services/discord"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/Jota-Music/jota/internal/services/youtube"
	"github.com/Jota-Music/jota/internal/update"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	ctx       context.Context
	version   string
	relay     Relay
	Spotify   *spotify.SpotifyService
	Catalog   *music.Catalog
	YouTube   *youtube.Service
	Sync      *rooms.Relay
	Follows   *follows.Service
	Rooms     *rooms.Store
	Order     *ordering.Service
	Playlists *playlists.Service
	Saved     *saved.Service
	Discord   *discord.Service
	// Backup is the document the library travels in, built from the stores
	// above and the settings bucket.
	Backup *backup.Service

	discordMu  sync.Mutex
	discord    *discordState
	discordErr string
	// syncMu serializes the passes that write the sync document: they all land on
	// the same temp file.
	syncMu sync.Mutex
	// syncQueued is the one background pass that has not started yet, and
	// syncRunning is the goroutine that takes it. Twenty follows in a row ask for
	// a pass each, but one is enough: they all read the same stores.
	syncQueued  atomic.Bool
	syncRunning atomic.Bool
}

// Relay is the endpoint the app dials: a URL and the token that relay wants.
type Relay struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

var installMu sync.Mutex

func New(version string) *App {
	cfg := env.Load()
	spotifySvc := spotify.NewSpotifyService(cfg.SpotifyClientID)
	// Late-bound like the relay emit below: Watch fires long after New returns,
	// so the application is resolvable by then.
	spotifySvc.OnReconnect(func() {
		if app := application.Get(); app != nil {
			app.Event.Emit("spotify:reconnected")
		}
	})
	youTubeSvc := youtube.NewService()
	playlistsSvc := playlists.New(nil)
	catalog := music.NewCatalog(spotifySvc, youTubeSvc, playlistsSvc)
	playlistsSvc.SetResolver(catalog.GetSong)
	playlistsSvc.SetTrackCache(spotifySvc.CachedSong, spotifySvc.SetCachedSong)
	followsSvc := follows.New()
	orderingSvc := ordering.New()
	savedSvc := saved.New()
	app := &App{
		version: version,
		relay: Relay{
			URL:   strings.TrimSpace(cfg.RelayAPIURL),
			Token: strings.TrimSpace(cfg.RelayAPIToken),
		},
		Spotify: spotifySvc,
		Catalog: catalog,
		YouTube: youTubeSvc,
		Sync: rooms.NewRelay(func(name string, data ...any) {
			app := application.Get()
			if app == nil {
				return
			}
			app.Event.Emit(name, data...)
		}),
		Follows:   followsSvc,
		Rooms:     rooms.NewStore(),
		Order:     orderingSvc,
		Playlists: playlistsSvc,
		Saved:     savedSvc,
		Discord:   discord.New(cfg.DiscordClientID),
		Backup: &backup.Service{
			Playlists: playlistsSvc,
			Follows:   followsSvc,
			Saved:     savedSvc,
			Order:     orderingSvc,
			Settings:  settingsSync,
		},
	}

	refresh.Register("spotify-catalog", spotifySvc.RefreshCatalog)
	refresh.Register("youtube-catalog", youTubeSvc.RefreshCatalog)

	return app
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	kv.Start()

	go func() {
		for {
			if runtime.GOOS != "android" && a.SyncAuto() && a.SyncFolder() != "" {
				if err := a.SyncPlaylists(); err != nil {
					log.Printf("sync: %v", err)
				}
			}
			time.Sleep(time.Duration(a.SyncInterval()) * time.Minute)
		}
	}()

	// Restoring the session is a network round trip, and startup runs before any
	// window exists: blocking here means the app never opens at all, which is
	// exactly how it used to hang on a stalled network. Restore in the
	// background, bounded; Watch retries a degraded session anyway.
	go func() {
		ctx, cancel := context.WithTimeout(ctx, spotify.SessionTimeout)
		defer cancel()
		if err := a.Spotify.Connect(ctx); err != nil {
			log.Printf("spotify: not connected at startup: %v", err)
			return
		}
		log.Printf("spotify: connected as %s", a.Spotify.Username())
	}()
	a.Spotify.Watch()
	refresh.Start(func() {
		app := application.Get()
		if app != nil {
			app.Event.Emit("refresh:updated")
		}
	})
	return nil
}

func (a *App) ServiceShutdown() error {
	a.Discord.Close()
	kv.Close()
	return nil
}

func (a *App) Version() string {
	return a.version
}

// LogError persists a frontend-reported error to the app log so failures that
// only surface in the UI stay diagnosable after the fact.
func (a *App) LogError(message string) {
	log.Printf("[ui] %s", message)
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
	if relaunchAfterUpdate() {
		go func() {
			time.Sleep(500 * time.Millisecond)
			application.Get().Quit()
		}()
	}
	return nil
}

// OpenURL opens the URL in the system browser. On Android it uses the app's
// native Intent.ACTION_VIEW so OAuth runs outside the WebView; on other
// platforms the platform opener (xdg-open, open, rundll32) handles it.
func (a *App) OpenURL(url string) error {
	return openExternal(url)
}

// RelayOverride reports the relay pinned through RELAY_API_URL/RELAY_API_TOKEN,
// if any. The frontend applies it over the user's saved relay so a dev build can
// point at a local relay.
func (a *App) RelayOverride() Relay {
	return a.relay
}

// RelayConfig reports the relay the user saved, if any. It lives in the store
// and not in the webview's localStorage because that storage belongs to the
// webview profile: a reinstall, a change of app id or a dev build on another
// origin all start from an empty one, and the relay is the first thing the user
// has to type again.
func (a *App) RelayConfig() Relay {
	var saved Relay
	if err := settings.GetObject("relay", &saved); err != nil {
		return Relay{}
	}
	return saved
}

// SaveRelay persists the relay the settings form shows. A relay pinned by the
// environment is not the user's to store, and an empty URL is not a relay, so
// both leave the saved one alone.
func (a *App) SaveRelay(url string, token string) error {
	if a.relay.URL != "" {
		return nil
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	if err := settings.SetObject("relay", Relay{
		URL:   url,
		Token: strings.TrimSpace(token),
	}); err != nil {
		return err
	}
	touch("relay")
	a.syncInBackground()
	return nil
}
