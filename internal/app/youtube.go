package app

import (
	"errors"
	"log"
	"runtime"
	"strings"

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
		if errors.Is(err, youtube.ErrLoginRequired) && !youtube.SignedIn() {
			application.Get().Event.Emit("youtube:signin-required", song.Name)
		}
		if errors.Is(err, youtube.ErrMembersOnly) && !youtube.SignedIn() {
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
		// A session YouTube rejects can never open the account's lists, so the
		// panel stops claiming it is signed in. A partial set is answered as a
		// rejected session by design (it only signs requests), and its verdict
		// would cost playback for nothing.
		if errors.Is(err, youtube.ErrSession) && youtube.SessionFull() {
			// The jar is where Google refreshes the session, and this is the
			// moment the user is looking at the shelf: asking for it here is
			// what keeps a working session from waiting for the next launch.
			if recoverYouTubeSession() {
				return []music.PlaylistSummary{}, nil
			}
			log.Printf("youtube: session YouTube rejected")
			expireYouTubeSession()
		}
		return []music.PlaylistSummary{}, nil
	}
	// The account's own lists answered, which is the evidence that the stored
	// session works: a mark from an earlier verdict stops applying.
	_ = youtube.MarkExpired(false)
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
// it; using an account this way can get it banned by Google. The verdict is the
// next signed request's, so an import is never refused for one bad answer; a set
// that only signs requests is refused locally (`ErrPartialSession`).
func (a *App) SetYouTubeCookies(cookies string) error {
	return a.storeYouTube(cookies, "")
}

// storeYouTube keeps a session and remembers which browser it came from: that is
// the jar a rejection is refreshed from. Nothing is asked of the network here: an
// import is never refused for an answer that could be one bad sample, and the next
// signed request is what tells whether the cookies work.
func (a *App) storeYouTube(cookies, browser string) error {
	if err := youtube.SetCookiesFrom(cookies, browser); err != nil {
		return err
	}
	if gaps := youtube.SessionGaps(youtube.Cookies()); len(gaps) > 0 {
		log.Printf("youtube: stored session signs requests only, missing %s", strings.Join(gaps, ", "))
	}
	a.syncYouTubeCookies()
	emitYouTubeSession()
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

// storeYouTubeSession decrypts a synced cookie blob with the local pass and
// keeps it as the account session, answering whether the stored session changed.
// A blob this device cannot open, and one that carries the cookies it already
// holds, are no-ops.
func storeYouTubeSession(blob string) bool {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return false
	}
	pass := storedPass()
	if pass == "" {
		return false
	}
	pt, err := decryptWithPass(pass, blob)
	if err != nil || pt == "" {
		return false
	}
	if youtube.Cookies() == pt {
		return false
	}
	localFull, incomingFull := youtube.Full(youtube.Cookies()), youtube.Full(pt)
	// A partial session must not overwrite a full one: the account's own
	// playlists stay closed with the partial set. Startup staging pushes the
	// full local session back to the document.
	if localFull && !incomingFull {
		return false
	}
	if err := youtube.SetCookies(pt); err != nil {
		return false
	}
	if gaps := youtube.SessionGaps(pt); len(gaps) > 0 {
		log.Printf("youtube: synced session applied but partial, missing %s", strings.Join(gaps, ", "))
	} else {
		log.Printf("youtube: synced session applied")
	}
	emitYouTubeSession()
	return true
}

// expireYouTubeSession marks the stored session as one YouTube answered signed
// out, so the panel stops claiming it works. The cookies stay: a rejected verdict
// can be one sample, and the copy in the document and in the browser's own jar is
// what it is refreshed from.
func expireYouTubeSession() {
	_ = youtube.MarkExpired(true)
	emitYouTubeSession()
}

// emitYouTubeSession tells the page the stored session changed, so the panel and
// the shelf read it again.
func emitYouTubeSession() {
	if app := application.Get(); app != nil {
		app.Event.Emit("youtube:session-updated")
	}
}

// The browser jar readers are variables so a test answers a session without this
// machine's browsers.
var (
	cookiesFromBrowsers = youtube.CookiesBrowsers
	browserCookies      = youtube.CookiesFromBrowser
)

// recoverYouTubeSession takes the session a browser on this machine holds, for a
// stored session YouTube answered as expired. It asks the network nothing: the
// request that failed is the evidence, and the copy is judged by the next one.
func recoverYouTubeSession() bool {
	stored := youtube.Cookies()
	remembered := youtube.SessionBrowser()
	// The jar the session came from is refreshed by the browser itself, so it is
	// asked first and without an account check: Google rotates these cookies, and
	// the one it hands back may well name another secret.
	if remembered != "" {
		if raw, err := browserCookies(remembered); err == nil && adoptYouTubeSession(remembered, raw, stored) {
			return true
		}
	}
	// Another browser continues the same session or it is not considered: signing
	// the device in as someone else costs more than asking for an import.
	for _, browser := range cookiesFromBrowsers() {
		if browser == remembered {
			continue
		}
		raw, err := browserCookies(browser)
		if err != nil || !youtube.SameAccount(stored, raw) {
			continue
		}
		if adoptYouTubeSession(browser, raw, stored) {
			return true
		}
	}
	return false
}

// adoptYouTubeSession keeps a jar's cookies over a rejected copy. A set this
// device already gave up on is not taken again, so a dead jar cannot keep the
// panel cycling between the same two answers.
func adoptYouTubeSession(browser, raw, dropped string) bool {
	if raw == "" || raw == dropped {
		return false
	}
	if err := youtube.SetCookiesFrom(raw, browser); err != nil {
		return false
	}
	log.Printf("youtube: session refreshed from %s", browser)
	stageYouTubeCookies()
	emitYouTubeSession()
	return true
}

// refreshYouTubeSession is the launch trigger, and it touches no network: the
// staged copy stands in when this device has none, and the browser's jar — the one
// the session came from, or one that continues the same account — keeps what it
// holds current. Evidence about a dead session comes from a real request, never
// from a probe here.
func refreshYouTubeSession() {
	if !youtube.HasAuth() {
		// The merge only applies a section that changed, so the copy the document
		// still holds is never asked for on its own: this is what brings back one
		// a previous launch dropped.
		storeYouTubeSession(stagedSession("youtube"))
		return
	}
	recoverYouTubeSession()
}

// stageYouTubeCookies seals the stored cookies for the next pass and answers
// whether it staged anything.
func stageYouTubeCookies() bool {
	pass := storedPass()
	if pass == "" {
		return false
	}
	raw := youtube.Cookies()
	if raw == "" {
		return false
	}
	// Staging the same plaintext again would re-encrypt it under a fresh salt
	// and mark the pass dirty for nothing. Only a blob this pass can open and
	// that matches is proof of identity; anything else stages as before.
	if stored := stagedSession("youtube"); stored != "" {
		if pt, err := decryptWithPass(pass, stored); err == nil && pt == raw {
			return false
		}
	}
	enc, err := encryptWithPass(pass, raw)
	if err != nil {
		return false
	}
	if err := stageBlob("youtube", enc); err != nil {
		return false
	}
	sessionsSync.dirty.Store(true)
	return true
}

// syncYouTubeCookies stages the stored cookies and asks for a pass when it did.
func (a *App) syncYouTubeCookies() {
	if stageYouTubeCookies() {
		a.syncInBackground()
	}
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
	if err := a.storeYouTube(raw, browser); err != nil {
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
	return youtube.SignedIn()
}

// YouTubeStatus is everything the panel and the shelf need about the stored
// session: signed in, whether it can open private resources, whether this
// platform can sign in from a browser, and who it belongs to.
type YouTubeStatus struct {
	SignedIn bool   `json:"signedIn"`
	Full     bool   `json:"full"`
	Browser  bool   `json:"browser"`
	Account  string `json:"account"`
}

// YouTubeStatus answers the four readings in one crossing of the bridge. They
// were four bindings, all of them asked before the first paint.
func (a *App) YouTubeStatus() YouTubeStatus {
	// The import fills the session this reads, and it runs in the background:
	// waiting here is two stats once the store is gone, and it is what keeps the
	// first launch after the upgrade from answering "signed out".
	importLegacy()
	return YouTubeStatus{
		SignedIn: youtube.SignedIn(),
		Full:     youtube.SessionFull(),
		Browser:  runtime.GOOS != "android",
		Account:  youtube.Account(),
	}
}

// YouTubeSessionFull reports whether the stored cookies can open private
// resources: the browser cannot read the ones that do, so a session started from
// the app window is always partial.
func (a *App) YouTubeSessionFull() bool {
	return youtube.SessionFull()
}
