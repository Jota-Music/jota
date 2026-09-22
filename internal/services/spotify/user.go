package spotify

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// GetUserProfile resolves a public profile (display name and avatar) for a
// Spotify username. The avatar endpoint is undocumented and may be unavailable;
// callers should treat an error or empty ImageURL as "no picture".
func (s *SpotifyService) GetUserProfile(username string) (music.UserProfile, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return music.UserProfile{}, nil
	}
	return kv.Cached(musicBucket, "user:"+strings.ToLower(username), func() (music.UserProfile, error) {
		return s.userProfile(username)
	})
}

func (s *SpotifyService) userProfile(username string) (music.UserProfile, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return music.UserProfile{}, ErrNotConnected
	}

	path := fmt.Sprintf("/user-profile-view/v3/profile/%s", url.PathEscape(username))
	var result struct {
		Name     string `json:"name"`
		ImageURL string `json:"image_url"`
		Images   []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := getJSON(ctx, sess, path, &result); err != nil {
		return music.UserProfile{}, err
	}

	profile := music.UserProfile{
		Username:    username,
		DisplayName: result.Name,
		ImageURL:    result.ImageURL,
	}
	if profile.ImageURL == "" && len(result.Images) > 0 {
		profile.ImageURL = result.Images[0].URL
	}
	if profile.DisplayName == "" {
		profile.DisplayName = username
	}
	return profile, nil
}
