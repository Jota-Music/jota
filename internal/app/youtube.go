package app

import (
	"errors"
	"log"
	"strings"
	"time"

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
		// A session YouTube rejects can never open the account's lists, so
		// drop it instead of leaving the panel saying it is signed in.
		if errors.Is(err, youtube.ErrSession) {
			log.Printf("youtube: clearing session YouTube rejected")
			clearYouTubeSession()
		}
		return []music.PlaylistSummary{}, nil
	}
	return list, nil
}

func (a *App) RevalidateYouTubeChannel(channel string) error {
	return a.YouTube.RevalidateChannel(channel)
}

// RevalidateMyYouTubePlaylists drops the cached list of the signed-in
// account's playlists, so a session applied from the sync document is followed
// by a fresh download on the next read.
func (a *App) RevalidateMyYouTubePlaylists() error {
	return a.YouTube.RevalidateMyYouTubePlaylists()
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
	a.syncYouTubeCookies()
	return nil
}

// exportYouTubeCookies returns the stored cookies as a portable string. When a
// passphrase is given they are encrypted so the file can be carried across
// devices without exposing the session.
func (a *App) exportYouTubeCookies(pass string) (string, error) {
	raw := youtube.Cookies()
	if raw == "" {
		return "", errors.New("no YouTube session stored")
	}
	return encryptWithPass(pass, raw)
}

// ImportYouTubeCookies accepts a portable string produced by exportYouTubeCookies
// (possibly encrypted) and stores it as the account session.
func (a *App) ImportYouTubeCookies(data, pass string) error {
	data = strings.TrimSpace(data)
	pt, err := decryptWithPass(pass, data)
	if err != nil {
		return err
	}
	return a.SetYouTubeCookies(pt)
}

// applyYouTubeCookies decrypts a synced cookie blob with the local pass and
// stores it as the account session. It is a no-op when there is no pass or
// the blob cannot be opened.
func applyYouTubeCookies(blob string) {
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
	localFull, incomingFull := youtube.Full(youtube.Cookies()), youtube.Full(pt)
	// A partial session must not overwrite a full one: the account's own
	// playlists stay closed with the partial set. Startup staging pushes the
	// full local session back to the document.
	if localFull && !incomingFull {
		return
	}
	if err := youtube.SetCookies(pt); err != nil {
		return
	}
	if app := application.Get(); app != nil {
		app.Event.Emit("youtube:session-updated")
	}
	go func() {
		if !errors.Is(youtube.Verify(), youtube.ErrSession) {
			return
		}
		log.Printf("youtube: clearing synced session YouTube rejected")
		clearYouTubeSession()
	}()
}

// clearYouTubeSession forgets a session YouTube rejected on this device alone, so
// the panel stops claiming it is signed in. It leaves the synced copy in place: a
// session this device cannot use may still work on another one, and only the user
// clearing it may delete it everywhere.
func clearYouTubeSession() {
	_ = youtube.ClearCookies()
	if app := application.Get(); app != nil {
		app.Event.Emit("youtube:session-updated")
	}
}

// dropYouTubeSession forgets the session everywhere: the panel stops claiming it
// is signed in and the tombstone removes it from the other devices. Only the user
// clearing the session asks for that.
func dropYouTubeSession() {
	_ = youtube.ClearCookies()
	tombstoneSession("youtube")
	if app := application.Get(); app != nil {
		app.Event.Emit("youtube:session-updated")
	}
}

// verifyYouTubeSession clears a stored session YouTube no longer accepts, so
// the panel stops claiming it is signed in and offers a fresh sign-in.
func verifyYouTubeSession() {
	if !youtube.HasAuth() {
		return
	}
	if !errors.Is(youtube.Verify(), youtube.ErrSession) {
		return
	}
	log.Printf("youtube: clearing stored session YouTube rejected")
	clearYouTubeSession()
}

// syncYouTubeCookies encrypts the stored cookies with the local pass and stages
// them for the next sync pass. It does nothing when there is no pass or no
// session.
func (a *App) syncYouTubeCookies() {
	pass := a.SessionPass()
	if pass == "" {
		return
	}
	raw := youtube.Cookies()
	if raw == "" {
		return
	}
	enc, err := encryptWithPass(pass, raw)
	if err != nil {
		return
	}
	if err := settings.SetObject("session:youtube", enc); err != nil {
		return
	}
	_ = settings.SetObject("session:youtube"+settingAt, time.Now().UnixMilli())
	sessionsSync.dirty.Store(true)
	a.syncInBackground()
}

func (a *App) ClearYouTubeCookies() error {
	err := youtube.ClearCookies()
	tombstoneSession("youtube")
	a.syncInBackground()
	return err
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
