package app

import (
	"errors"
	"log"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
)

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

func (a *App) ResolveYouTubeId(song music.Song) (string, error) {
	return a.YouTube.Find(song)
}

func (a *App) SetYouTubeId(cacheKey string, youtubeId string) error {
	return a.YouTube.SetYoutubeId(cacheKey, youtubeId)
}

func (a *App) GetYouTubeId(cacheKey string) string {
	return a.YouTube.YoutubeId(cacheKey)
}

func (a *App) SearchYouTube(query string) ([]youtube.Video, error) {
	return a.YouTube.Search(query)
}

func (a *App) SearchYouTubePlaylists(query string) ([]music.PlaylistSummary, error) {
	return a.YouTube.SearchPlaylists(query)
}

func (a *App) SearchYouTubeChannels(query string) ([]music.ChannelInfo, error) {
	return a.YouTube.SearchChannels(query)
}

func (a *App) GetYouTubeChannelPlaylists(channel string) ([]music.PlaylistSummary, error) {
	return a.YouTube.GetChannelPlaylists(channel)
}

// GetMyYouTubePlaylists returns the playlists of the signed-in YouTube account.
// A session YouTube no longer accepts is answered as an empty list, not as an
// error: this feeds a shelf next to Spotify's and the saved library, so a stale
// session must only cost its own source, not the view. The reason is logged, and
// the account panel's refresh retries it.
func (a *App) GetMyYouTubePlaylists() ([]music.PlaylistSummary, error) {
	list, err := a.YouTube.GetMyYouTubePlaylists()
	if err != nil {
		log.Printf("youtube: account playlists: %v", err)
		return []music.PlaylistSummary{}, nil
	}
	return list, nil
}

func (a *App) RevalidateYouTubeChannel(channel string) error {
	return a.YouTube.RevalidateChannel(channel)
}

func (a *App) GetYouTubeChannelVideos(channel string) ([]music.Song, error) {
	return a.YouTube.GetChannelVideos(channel)
}

func (a *App) GetYouTubeChannelInfo(channel string) (music.ChannelInfo, error) {
	return a.YouTube.GetChannelInfo(channel)
}

// SetYouTubeCookies stores the Google account cookies used to sign innertube
// requests. Playing age-restricted videos and avoiding the bot check requires
// it; using an account this way can get it banned by Google. The session is
// checked against YouTube before it is kept, because a partial export signs
// requests yet is still anonymous, which only shows up later on a private
// playlist.
func (a *App) SetYouTubeCookies(cookies string) error {
	if err := youtube.SetCookies(cookies); err != nil {
		return err
	}
	if err := youtube.Verify(); err != nil {
		_ = youtube.ClearCookies()
		return err
	}
	return nil
}

func (a *App) ClearYouTubeCookies() error {
	return youtube.ClearCookies()
}

// CookiesBrowsers lists the browsers on this machine whose cookie jar holds a
// YouTube session, so the desktop UI only offers a browser import it can
// actually perform. Empty on platforms without a jar reader.
func (a *App) CookiesBrowsers() []string {
	return youtube.CookiesBrowsers()
}

// YouTubeBrowserAccounts labels each browser whose jar can be read with the
// account it holds, so the panel offers the right one instead of an opaque name.
func (a *App) YouTubeBrowserAccounts() map[string]string {
	accounts := make(map[string]string)
	for _, b := range youtube.BrowserAccounts() {
		accounts[b.Name] = b.Account
	}
	return accounts
}

// YouTubeCookiesFromBrowser signs in with the session a browser on this machine
// already holds, so the user never pastes a cookie header. The jar is read and
// verified here, not in the UI, and the account's playlists are refetched only
// once YouTube accepted the session.
func (a *App) YouTubeCookiesFromBrowser(browser string) error {
	raw, err := youtube.CookiesFromBrowser(browser)
	if err != nil {
		return err
	}
	if err := a.SetYouTubeCookies(raw); err != nil {
		return err
	}
	return a.YouTube.RevalidateMyYouTubePlaylists()
}

// YouTubeAccount names the account the stored session belongs to, so the panel
// can say who is connected instead of only that someone is. Empty when the
// session is one YouTube does not recognise (the partial set the sign-in window
// can read).
func (a *App) YouTubeAccount() string {
	return youtube.Account()
}

func (a *App) YouTubeSignedIn() bool {
	return youtube.HasAuth()
}

// YouTubeSessionFull reports whether the stored cookies can open private
// resources: the browser cannot read the ones that do, so a session started from
// the app window is always partial.
func (a *App) YouTubeSessionFull() bool {
	return youtube.SessionFull()
}
