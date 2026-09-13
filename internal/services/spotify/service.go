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

	"jota/server/internal/kv"

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

	clientID       string
	callbackServer *callbackServer
	pendingMu      sync.Mutex
	pending        *pendingLogin
}

type pendingLogin struct {
	verifier    string
	redirectURL string
	createdAt   time.Time
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
	var lastErr error
	for attempt := 1; attempt <= 8; attempt++ {
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
		log.Printf("spotify-v2: session creation failed (attempt %d/8): %v", attempt, err)
		time.Sleep(3 * time.Second)
	}
	return nil, lastErr
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
	return s.Connect(ctx)
}

func (s *SpotifyService) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != nil {
		s.sess.Close()
		s.sess = nil
	}
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
	s.mu.Unlock()

	cbServer, err := newCallbackServer()
	if err != nil {
		log.Printf("spotify-v2: StartupLogin failed: %v", err)
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}
	s.callbackServer = cbServer
	log.Printf("spotify-v2: callback server started on port %d", cbServer.port)
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/login", cbServer.port)

	oauthConf := &oauth2.Config{
		ClientID:    s.clientID,
		RedirectURL: redirectURL,
		Scopes:      spotifyOAuthScopes,
		Endpoint:    spotifyoauth2.Endpoint,
	}

	verifier := oauth2.GenerateVerifier()
	authURL := oauthConf.AuthCodeURL("", oauth2.S256ChallengeOption(verifier))

	p := &pendingLogin{
		verifier:    verifier,
		redirectURL: redirectURL,
		createdAt:   time.Now(),
	}
	s.pendingMu.Lock()
	s.pending = p
	s.pendingMu.Unlock()

	log.Printf("spotify-v2: auth URL: %s", authURL)

	go func() {
		time.Sleep(loginTimeout)
		s.pendingMu.Lock()
		if s.pending == p {
			s.pending = nil
			s.callbackServer.stop()
			log.Printf("spotify-v2: login timeout")
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
		log.Printf("spotify-v2: CompleteLogin: no pending login")
		return ErrNoLoginInProgress
	}
	if s.callbackServer == nil {
		log.Printf("spotify-v2: CompleteLogin: no callback server")
		return errors.New("no callback server running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	log.Printf("spotify-v2: waiting for callback...")
	code, err := s.callbackServer.wait(ctx)
	s.callbackServer.stop()
	s.callbackServer = nil
	if err != nil {
		log.Printf("spotify-v2: callback error: %v", err)
		return fmt.Errorf("callback wait: %w", err)
	}
	if code == "" {
		log.Printf("spotify-v2: no code received")
		return errors.New("no code received from Spotify")
	}
	log.Printf("spotify-v2: got code")

	log.Printf("spotify-v2: completing interactive login")
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
		log.Printf("spotify-v2: token exchange failed (attempt %d/3): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed exchanging oauth2 code: %w", err)
	}
	log.Printf("spotify-v2: token exchanged")

	username, _ := token.Extra("username").(string)
	if username == "" {
		log.Printf("spotify-v2: missing username in token")
		return errors.New("missing username in token response")
	}

	sess, err := s.newSession(context.Background(), session.SpotifyTokenCredentials{
		Username: username,
		Token:    token.AccessToken,
	})
	if err != nil {
		log.Printf("spotify-v2: session creation failed: %v", err)
		return fmt.Errorf("failed connecting session: %w", err)
	}

	s.mu.Lock()
	s.sess = sess
	s.mu.Unlock()

	if err := saveCreds(sess.Username(), sess.StoredCredentials()); err != nil {
		log.Printf("spotify-v2: failed to save credentials: %v", err)
	}
	log.Printf("spotify-v2: connected as %s", sess.Username())

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

func mustHexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		log.Fatalf("spotify-v2: invalid stored credentials (not hex): %v", err)
	}
	return b
}

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
	}
}

func (l *browserLogger) Trace(...interface{})                           {}
func (l *browserLogger) Debug(...interface{})                           {}
func (l *browserLogger) Info(...interface{})                            {}
func (l *browserLogger) Warn(...interface{})                            {}
func (l *browserLogger) Error(...interface{})                           {}
func (l *browserLogger) WithField(string, interface{}) librespot.Logger { return l }
func (l *browserLogger) WithError(error) librespot.Logger               { return l }

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
