package app

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/env"
	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/refresh"
	"github.com/Jota-Music/jota/internal/rooms"
	"github.com/Jota-Music/jota/internal/saved"
	"github.com/Jota-Music/jota/internal/services/discord"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/Jota-Music/jota/internal/services/youtube"
	"github.com/Jota-Music/jota/internal/update"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	ctx       context.Context
	version   string
	relay     RelayOverride
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

	discordMu  sync.Mutex
	discord    *discordState
	discordErr string
}

// RelayOverride pins a relay for local testing. An empty URL means the user's
// saved relay is used.
type RelayOverride struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

var installMu sync.Mutex

func New(version string) *App {
	cfg := env.Load()
	spotifySvc := spotify.NewSpotifyService(cfg.SpotifyClientID)
	youTubeSvc := youtube.NewService()
	playlistsSvc := playlists.New(nil)
	catalog := music.NewCatalog(spotifySvc, youTubeSvc, playlistsSvc)
	playlistsSvc.SetResolver(catalog.GetSong)
	app := &App{
		version: version,
		relay: RelayOverride{
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
		Follows:   follows.New(),
		Rooms:     rooms.NewStore(),
		Order:     ordering.New(),
		Playlists: playlistsSvc,
		Saved:     saved.New(),
		Discord:   discord.New(cfg.DiscordClientID),
	}

	refresh.Register("spotify-catalog", spotifySvc.RefreshCatalog)
	refresh.Register("youtube-catalog", youTubeSvc.RefreshCatalog)

	return app
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	kv.Start()
	if err := a.Spotify.Connect(ctx); err != nil {
		log.Printf("spotify: not connected at startup: %v", err)
	} else {
		log.Printf("spotify: connected as %s", a.Spotify.Username())
	}
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
	go func() {
		time.Sleep(500 * time.Millisecond)
		application.Get().Quit()
	}()
	return nil
}

// OpenURL opens the URL in the system browser. On Android it uses the app's
// native Intent.ACTION_VIEW so OAuth runs outside the WebView; on other
// platforms the frontend opens the URL itself.
func (a *App) OpenURL(url string) error {
	return openExternal(url)
}

// RelayOverride reports the relay pinned through RELAY_API_URL/RELAY_API_TOKEN,
// if any. The frontend applies it over the user's saved relay so a dev build can
// point at a local relay.
func (a *App) RelayOverride() RelayOverride {
	return a.relay
}
