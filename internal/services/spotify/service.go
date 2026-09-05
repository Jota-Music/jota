package spotify

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"jota/server/internal/kv"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
	"golang.org/x/oauth2"
	spotifyoauth2 "golang.org/x/oauth2/spotify"
)

const (
	credsKey     = "global"
	bucketName   = "spotify-global"
	callbackPort = 8090
	loginTimeout = 5 * time.Minute
)

var sessionBucket = kv.UseBucket(bucketName)

type storedCreds struct {
	Username string `json:"username"`
	Data     string `json:"data"`
}

type SpotifyService struct {
	mu     sync.Mutex
	sess   *session.Session
	done   bool

	pendingMu sync.Mutex
	pending   *pendingLogin
}

type pendingLogin struct {
	verifier    string
	origin      string
	redirectURL string
	createdAt   time.Time
}

func NewSpotifyService() *SpotifyService {
	return &SpotifyService{}
}

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
		log.Printf("spotify-v2: stored credentials found for %s", creds.Username)
		sess, err := s.newSession(ctx, session.StoredCredentials{Username: creds.Username, Data: mustHexDecode(creds.Data)})
		if err == nil {
			s.sess = sess
			log.Printf("spotify-v2: connected as %s", sess.Username())
			return nil
		}
		log.Printf("spotify-v2: failed to restore session: %v", err)
	}

	log.Printf("spotify-v2: no stored credentials (%v), serving disconnected; log in from the app", err)
	return ErrNotConnected
}

func (s *SpotifyService) newSession(ctx context.Context, creds any) (*session.Session, error) {
	return session.NewSessionFromOptions(ctx, &session.Options{
		Log:         &browserLogger{},
		DeviceType:  devicespb.DeviceType_COMPUTER,
		DeviceId:    "0123456789abcdef0123456789abcdef01234567",
		Credentials: creds,
	})
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
		return ""
	}
	return s.sess.Username()
}

func (s *SpotifyService) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sess != nil
}

func (s *SpotifyService) Reconnect(ctx context.Context) error {
	s.mu.Lock()
	if s.sess != nil {
		s.sess.Close()
		s.sess = nil
	}
	s.done = false
	s.mu.Unlock()
	return s.Connect(ctx)
}

func (s *SpotifyService) Disconnect() error {
	s.mu.Lock()
	if s.sess != nil {
		s.sess.Close()
		s.sess = nil
	}
	s.done = true
	s.mu.Unlock()
	return sessionBucket.Delete(credsKey)
}

// StartInteractiveLogin closes any existing session and returns the Spotify
// authorize URL. The caller is expected to send the user there; once Spotify
// redirects back to /login with a code, ResolveLogin completes the login.
func (s *SpotifyService) StartInteractiveLogin(redirectOrigin, callbackPort string) (string, error) {
	s.mu.Lock()
	if s.sess != nil {
		s.sess.Close()
		s.sess = nil
	}
	s.done = false
	s.mu.Unlock()

	oauthConf := &oauth2.Config{
		ClientID:    librespot.ClientIdHex,
		RedirectURL: fmt.Sprintf("http://127.0.0.1:%s/login", callbackPort),
		Scopes:      spotifyOAuthScopes,
		Endpoint:    spotifyoauth2.Endpoint,
	}

	verifier := oauth2.GenerateVerifier()
	url := oauthConf.AuthCodeURL("", oauth2.S256ChallengeOption(verifier))

	p := &pendingLogin{
		verifier:    verifier,
		origin:      redirectOrigin,
		redirectURL: oauthConf.RedirectURL,
		createdAt:   time.Now(),
	}
	s.pendingMu.Lock()
	s.pending = p
	s.pendingMu.Unlock()

	go func() {
		time.Sleep(loginTimeout)
		s.pendingMu.Lock()
		if s.pending == p {
			s.pending = nil
		}
		s.pendingMu.Unlock()
	}()

	return url, nil
}

var ErrNoLoginInProgress = errors.New("no login in progress")

// ResolveLogin completes a pending interactive login with the given OAuth code.
// It runs synchronously so the caller can show a clear success/failure page.
// It returns the app origin to redirect the browser back to.
func (s *SpotifyService) ResolveLogin(code string) (string, error) {
	s.pendingMu.Lock()
	p := s.pending
	s.pending = nil
	s.pendingMu.Unlock()
	if p == nil {
		return "", ErrNoLoginInProgress
	}

	log.Printf("spotify-v2: completing interactive login")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	oauthConf := &oauth2.Config{
		ClientID:    librespot.ClientIdHex,
		RedirectURL: p.redirectURL,
		Scopes:      spotifyOAuthScopes,
		Endpoint:    spotifyoauth2.Endpoint,
	}

	token, err := oauthConf.Exchange(ctx, code, oauth2.VerifierOption(p.verifier))
	if err != nil {
		return p.origin, fmt.Errorf("failed exchanging oauth2 code: %w", err)
	}

	username, _ := token.Extra("username").(string)
	if username == "" {
		return p.origin, errors.New("missing username in token response")
	}

	sess, err := s.newSession(context.Background(), session.SpotifyTokenCredentials{
		Username: username,
		Token:    token.AccessToken,
	})
	if err != nil {
		return p.origin, fmt.Errorf("failed connecting session: %w", err)
	}

	s.mu.Lock()
	s.sess = sess
	s.mu.Unlock()

	if err := saveCreds(sess.Username(), sess.StoredCredentials()); err != nil {
		log.Printf("spotify-v2: failed to save credentials: %v", err)
	}
	log.Printf("spotify-v2: connected as %s", sess.Username())

	return p.origin, nil
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

func mustHexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		log.Fatalf("spotify-v2: invalid stored credentials (not hex): %v", err)
	}
	return b
}

var authURLPattern = regexp.MustCompile(`https?://[^\s]+`)

type browserLogger struct{}

func (l *browserLogger) Tracef(string, ...interface{}) {}
func (l *browserLogger) Debugf(string, ...interface{}) {}
func (l *browserLogger) Warnf(string, ...interface{})  {}
func (l *browserLogger) Errorf(string, ...interface{}) {}

func (l *browserLogger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)
	if url := authURLPattern.FindString(msg); url != "" {
		fmt.Printf("\n>>> Open this URL in your browser to log in: %s\n\n", url)
		_ = exec.Command("xdg-open", url).Start()
	}
}

func (l *browserLogger) Trace(...interface{})                  {}
func (l *browserLogger) Debug(...interface{})                  {}
func (l *browserLogger) Info(...interface{})                   {}
func (l *browserLogger) Warn(...interface{})                   {}
func (l *browserLogger) Error(...interface{})                  {}
func (l *browserLogger) WithField(string, interface{}) librespot.Logger { return l }
func (l *browserLogger) WithError(error) librespot.Logger            { return l }

var ErrNotConnected = errors.New("spotify is not connected")