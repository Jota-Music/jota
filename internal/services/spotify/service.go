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

	"jota/server/internal/kv"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
)

const (
	credsKey     = "global"
	bucketName   = "spotify-global"
	callbackPort = 8090
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
}

func NewSpotifyService() *SpotifyService {
	return &SpotifyService{}
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

	log.Printf("spotify-v2: no stored credentials, opening browser for Spotify login on port %d", callbackPort)

	sess, err := s.newSession(ctx, session.InteractiveCredentials{CallbackPort: callbackPort})
	if err != nil {
		s.done = true
		return fmt.Errorf("interactive login: %w", err)
	}

	blob := sess.StoredCredentials()
	if err := saveCreds(sess.Username(), blob); err != nil {
		log.Printf("spotify-v2: failed to save credentials: %v", err)
	}

	s.sess = sess
	log.Printf("spotify-v2: connected as %s", sess.Username())
	return nil
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