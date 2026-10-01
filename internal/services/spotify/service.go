package spotify

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/kv"

	librespot "github.com/devgianlu/go-librespot"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
	"github.com/devgianlu/go-librespot/session"
	"golang.org/x/oauth2"
	spotifyoauth2 "golang.org/x/oauth2/spotify"
)

const (
	credsKey     = "global"
	bucketName   = "spotify-global"
	loginTimeout = 5 * time.Minute

	// restoreAttempts is how often Connect retries a stored session. Restoring
	// is best-effort: the watcher retries in the background, so blocking
	// startup here only delays the UI.
	restoreAttempts = 1
	// reconnectTimeout bounds one background attempt so a blackholed network
	// cannot park the watcher forever.
	reconnectTimeout = 10 * time.Second
)

var sessionBucket = kv.UseBucket(bucketName)

type storedCreds struct {
	Username string `json:"username"`
	Data     string `json:"data"`
}

type SpotifyService struct {
	mu   sync.Mutex
	sess *session.Session
	done bool

	// username survives a failed restore so a degraded service still reads the
	// cache written for this account instead of the "default" bucket.
	username string
	degraded bool
	watching bool

	clientID  string
	pendingMu sync.Mutex
	pending   *pendingLogin

	// onReconnect fires when the watcher brings a degraded service back.
	onReconnect func()
}

type pendingLogin struct {
	verifier    string
	redirectURL string
	server      *callbackServer
}

func NewSpotifyService(clientID string) *SpotifyService {
	if clientID == "" {
		clientID = librespot.ClientIdHex
	}
	return &SpotifyService{clientID: clientID}
}

func (s *SpotifyService) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sess != nil {
		return nil
	}
	if s.done {
		return ErrNotConnected
	}

	creds, err := loadCreds()
	if err == nil {
		data, decodeErr := hex.DecodeString(creds.Data)
		if decodeErr != nil {
			log.Printf("spotify: invalid stored credentials (not hex): %v", decodeErr)
			_ = sessionBucket.Delete(credsKey)
			return ErrNotConnected
		}
		s.username = creds.Username
		log.Printf("spotify: stored credentials found for %s", creds.Username)
		sess, err := s.newSession(ctx, session.StoredCredentials{Username: creds.Username, Data: data}, restoreAttempts)
		if err == nil {
			s.sess = sess
			s.degraded = false
			log.Printf("spotify: connected as %s", sess.Username())
			return nil
		}
		// Spotify is unreachable but we have an account, so the cached catalog
		// is still usable. Serve it and let Watch retry in the background.
		s.degraded = true
		log.Printf("spotify: failed to restore session: %v", err)
		return ErrNotConnected
	}

	log.Printf("spotify: no stored credentials (%v), serving disconnected; log in from the app", err)
	return ErrNotConnected
}

// Watch retries a degraded restore until it succeeds, then reports it through
// onReconnect. It is a no-op while connected and exits on Disconnect.
func (s *SpotifyService) Watch() {
	s.mu.Lock()
	if s.watching {
		s.mu.Unlock()
		return
	}
	s.watching = true
	s.mu.Unlock()

	go s.reconnectLoop()
}

func (s *SpotifyService) reconnectLoop() {
	// ponytail: fixed ladder 15s -> 1m -> 4m -> 5m, no jitter; a real fleet
	// would want backoff with jitter.
	wait := 15 * time.Second
	for {
		time.Sleep(wait)
		if next := wait * 4; next < 5*time.Minute {
			wait = next
		} else {
			wait = 5 * time.Minute
		}
		// degraded is the only reason to keep trying. It clears on an
		// interactive login, which stops the watcher from restoring the old
		// session underneath CompleteLogin.
		if s.IsConnected() || !s.IsDegraded() || s.isFinished() {
			return
		}
		if !s.Reconnect() {
			log.Printf("spotify: reconnect attempt failed")
			continue
		}
		log.Printf("spotify: reconnected after outage")
		if s.onReconnect != nil {
			s.onReconnect()
		}
		return
	}
}

func (s *SpotifyService) newSession(ctx context.Context, creds any, attempts int) (*session.Session, error) {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		sess, err := session.NewSessionFromOptions(ctx, &session.Options{
			Log:         &browserLogger{},
			DeviceType:  devicespb.DeviceType_COMPUTER,
			DeviceId:    "0123456789abcdef0123456789abcdef01234567",
			Credentials: creds,
		})
		if err == nil {
			return sess, nil
		}
		lastErr = err
		log.Printf("spotify: session creation failed (attempt %d/%d): %v", attempt, attempts, err)
		if attempt < attempts {
			time.Sleep(3 * time.Second)
		}
	}
	return nil, lastErr
}

// OnReconnect registers the callback fired when Watch recovers the session.
func (s *SpotifyService) OnReconnect(fn func()) {
	s.onReconnect = fn
}

// Reconnect retries the restore with the background timeout applied, so no
// caller can block on a blackholed network. It reports whether the session is
// live again.
func (s *SpotifyService) Reconnect() bool {
	ctx, cancel := context.WithTimeout(context.Background(), reconnectTimeout)
	defer cancel()
	return s.Connect(ctx) == nil
}

func (s *SpotifyService) Session() *session.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sess
}

func (s *SpotifyService) Username() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess == nil {
		return s.username
	}
	return s.sess.Username()
}

func (s *SpotifyService) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sess != nil
}

// IsDegraded reports a stored account whose session could not be restored.
// The cached catalog still works, unlike IsConnected's live-session requirement.
func (s *SpotifyService) IsDegraded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.degraded
}

func (s *SpotifyService) isFinished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

func (s *SpotifyService) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != nil {
		s.sess.Close()
		s.sess = nil
	}
	s.username = ""
	s.degraded = false
	s.done = true
	return sessionBucket.Delete(credsKey)
}

func (s *SpotifyService) StartupLogin() (string, error) {
	s.mu.Lock()
	if s.sess != nil {
		s.sess.Close()
		s.sess = nil
	}
	s.done = false
	s.degraded = false
	s.mu.Unlock()

	verifier := oauth2.GenerateVerifier()
	// The state is only ever compared against itself on the loopback callback,
	// so it just has to be unpredictable. It ties the redirect to this request
	// and keeps a stray hit on the port from delivering a foreign code.
	state := oauth2.GenerateVerifier()
	cbServer, err := newCallbackServer(state)
	if err != nil {
		log.Printf("spotify: StartupLogin failed: %v", err)
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}
	log.Printf("spotify: callback server started on port %d", cbServer.port)
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/login", cbServer.port)

	oauthConf := &oauth2.Config{
		ClientID:    s.clientID,
		RedirectURL: redirectURL,
		Scopes:      spotifyOAuthScopes,
		Endpoint:    spotifyoauth2.Endpoint,
	}

	authURL := oauthConf.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	p := &pendingLogin{
		verifier:    verifier,
		redirectURL: redirectURL,
		server:      cbServer,
	}
	s.pendingMu.Lock()
	s.pending = p
	s.pendingMu.Unlock()

	log.Printf("spotify: auth URL: %s", authURL)

	go func() {
		time.Sleep(loginTimeout)
		s.pendingMu.Lock()
		if s.pending == p {
			s.pending = nil
			p.server.stop()
			log.Printf("spotify: login timeout")
		}
		s.pendingMu.Unlock()
	}()

	return authURL, nil
}

var ErrNoLoginInProgress = errors.New("no login in progress")

func (s *SpotifyService) CompleteLogin() error {
	s.pendingMu.Lock()
	p := s.pending
	s.pending = nil
	s.pendingMu.Unlock()
	if p == nil {
		log.Printf("spotify: CompleteLogin: no pending login")
		return ErrNoLoginInProgress
	}
	if p.server == nil {
		log.Printf("spotify: CompleteLogin: no callback server")
		return errors.New("no callback server running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	log.Printf("spotify: waiting for callback...")
	code, err := p.server.wait(ctx)
	p.server.stop()
	if err != nil {
		log.Printf("spotify: callback error: %v", err)
		return fmt.Errorf("callback wait: %w", err)
	}
	if code == "" {
		log.Printf("spotify: no code received")
		return errors.New("no code received from Spotify")
	}
	log.Printf("spotify: got code")

	log.Printf("spotify: completing interactive login")
	exchangeCtx, exchangeCancel := context.WithTimeout(context.Background(), time.Minute)
	defer exchangeCancel()

	oauthConf := &oauth2.Config{
		ClientID:    s.clientID,
		RedirectURL: p.redirectURL,
		Scopes:      spotifyOAuthScopes,
		Endpoint:    spotifyoauth2.Endpoint,
	}

	var token *oauth2.Token
	for attempt := 1; attempt <= 3; attempt++ {
		token, err = oauthConf.Exchange(exchangeCtx, code, oauth2.VerifierOption(p.verifier))
		if err == nil {
			break
		}
		log.Printf("spotify: token exchange failed (attempt %d/3): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed exchanging oauth2 code: %w", err)
	}
	log.Printf("spotify: token exchanged")

	username, _ := token.Extra("username").(string)
	if username == "" {
		log.Printf("spotify: missing username in token")
		return errors.New("missing username in token response")
	}

	sess, err := s.newSession(context.Background(), session.SpotifyTokenCredentials{
		Username: username,
		Token:    token.AccessToken,
	}, 8)
	if err != nil {
		log.Printf("spotify: session creation failed: %v", err)
		return fmt.Errorf("failed connecting session: %w", err)
	}

	s.mu.Lock()
	s.sess = sess
	s.username = sess.Username()
	s.degraded = false
	s.mu.Unlock()

	if err := saveCreds(sess.Username(), sess.StoredCredentials()); err != nil {
		log.Printf("spotify: failed to save credentials: %v", err)
	}
	log.Printf("spotify: connected as %s", sess.Username())

	return nil
}

func loadCreds() (*storedCreds, error) {
	var sc storedCreds
	err := sessionBucket.GetObject(credsKey, &sc)
	if err != nil {
		return nil, err
	}
	if sc.Username == "" || sc.Data == "" {
		return nil, errors.New("empty stored credentials")
	}
	return &sc, nil
}

func saveCreds(username string, data []byte) error {
	return sessionBucket.SetObject(credsKey, &storedCreds{
		Username: username,
		Data:     hex.EncodeToString(data),
	})
}

type browserLogger struct {
	librespot.NullLogger
}

func (l *browserLogger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)
	if url := authURLPattern.FindString(msg); url != "" {
		fmt.Printf("\n>>> Open this URL in your browser to log in: %s\n\n", url)
	}
}

var authURLPattern = regexp.MustCompile(`https://[^\s]+`)
var ErrNotConnected = errors.New("spotify is not connected")

var spotifyOAuthScopes = []string{
	"app-remote-control",
	"playlist-modify",
	"playlist-modify-private",
	"playlist-modify-public",
	"playlist-read",
	"playlist-read-collaborative",
	"playlist-read-private",
	"streaming",
	"ugc-image-upload",
	"user-follow-modify",
	"user-follow-read",
	"user-library-modify",
	"user-library-read",
	"user-modify",
	"user-modify-playback-state",
	"user-modify-private",
	"user-personalized",
	"user-read-birthdate",
	"user-read-currently-playing",
	"user-read-email",
	"user-read-play-history",
	"user-read-playback-position",
	"user-read-playback-state",
	"user-read-private",
	"user-read-recently-played",
	"user-top-read",
}
