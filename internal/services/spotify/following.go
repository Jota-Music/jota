package spotify

import (
	"context"
	"net/url"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var followPrefixes = []struct {
	prefix string
	kind   string
}{
	{"spotify:artist:", "artist"},
	{"spotify:user:", "user"},
}

// GetFollowing returns the profiles the account follows. The profile-view
// endpoint exposes the artist collection (and any user rows the service
// includes); the user-only follow graph is protobuf and not queried here.
//
// Cached like the rest of the catalog: it is a shelf the home page reads on
// every load, and it should survive a Spotify outage.
func (s *SpotifyService) GetFollowing() ([]music.Follow, error) {
	username := s.Username()
	if username == "" {
		return nil, ErrNotConnected
	}
	return kv.Cached(musicBucket, followingKey(username), s.follows)
}

func (s *SpotifyService) RevalidateFollowing() {
	username := s.Username()
	if username == "" {
		return
	}
	kv.Refreshable(musicBucket, followingKey(username), s.follows)
}

func followingKey(username string) string {
	return "following:v1:" + username
}

func (s *SpotifyService) follows() ([]music.Follow, error) {
	ctx := context.Background()
	sess := s.Session()
	username := s.Username()
	if sess == nil || username == "" {
		return nil, ErrNotConnected
	}

	path := "/user-profile-view/v3/profile/" + url.PathEscape(username) + "/following"
	var result struct {
		Profiles []struct {
			URI      string `json:"uri"`
			Name     string `json:"name"`
			ImageURL string `json:"image_url"`
		} `json:"profiles"`
	}
	if err := getJSON(ctx, sess, path, &result); err != nil {
		return nil, err
	}

	follows := make([]music.Follow, 0, len(result.Profiles))
	for _, p := range result.Profiles {
		for _, f := range followPrefixes {
			id := strings.TrimPrefix(p.URI, f.prefix)
			if id == p.URI || id == "" {
				continue
			}
			follows = append(follows, music.Follow{
				Id:       id,
				Name:     p.Name,
				ImageURL: p.ImageURL,
				Kind:     f.kind,
			})
			break
		}
	}
	return follows, nil
}
