package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type userToken struct {
	ClientId                         string `json:"clientId"`
	AccessToken                      string `json:"accessToken"`
	AccessTokenExpirationTimestampMs int64  `json:"accessTokenExpirationTimestampMs"`
	IsAnonymous                      bool   `json:"isAnonymous"`
	Notes                            string `json:"_notes,omitempty"`
}

func (s *SpotifyService) GetToken() (*userToken, error) {
	if cached, ok := loadTokenFromCache(); ok {
		s.token = cached
		return cached, nil
	}

	// Cache miss or expired: do not reuse in-memory token without the same validity
	// checks, or Spotify returns 401 and the playlist handler surfaces 500.
	if accessTokenValid(s.token) {
		return s.token, nil
	}
	s.token = nil

	token, err := s.getTokenFromRemote()
	if err != nil {
		return nil, err
	}

	s.token = token
	if err := saveTokenToCache(*token); err != nil {
		log.Printf("spotify failed to write token cache '%v'", err)
	}

	return s.token, nil
}

// invalidateToken drops the in-memory bearer and the persisted copy so the next
// GetToken must fetch a new web token (e.g. after Spotify returns 401).
func (s *SpotifyService) invalidateToken() {
	s.token = nil
	if err := clearTokenCache(); err != nil {
		log.Printf("spotify: clear token cache: %v", err)
	}
}

func (s *SpotifyService) getTokenFromRemote() (*userToken, error) {
	u := launcher.New().
		Leakless(false).
		Headless(true).
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://open.spotify.com")

	router := page.HijackRequests()
	defer router.MustStop()

	done := make(chan string, 1)

	router.MustAdd("*/api/token*", func(ctx *rod.Hijack) {
		ctx.MustLoadResponse()

		if strings.Contains(ctx.Request.URL().String(), "/api/token") {
			select {
			case done <- string(ctx.Response.Body()):
			default:
			}
		}
	})

	go router.Run()

	page.MustWaitLoad()

	select {
	case body := <-done:
		if body == "" {
			return nil, errors.New("empty token response")
		}

		var resp userToken
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			return nil, fmt.Errorf("failed to parse token response '%s' %w", err.Error(), err)
		}

		return &resp, nil

	case <-time.After(20 * time.Second):
		return nil, errors.New("timeout waiting for token")
	}
}
