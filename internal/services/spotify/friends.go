package spotify

import (
	"context"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
)

// GetFriends returns the usernames in the account's buddy list (the friend
// activity feed). The endpoint is undocumented and may be unavailable; callers
// should treat an error as "no friends to show". The answer is cached per account
// like the follows are: the shelf that shows it waits for this call, and a list
// that only changes on the other side does not need a round trip on every visit.
func (s *SpotifyService) GetFriends() ([]string, error) {
	username := s.Username()
	if username == "" {
		return nil, ErrNotConnected
	}
	return kv.Cached(musicBucket, friendsKey(username), s.friends)
}

func friendsKey(username string) string {
	return "friends:v1:" + username
}

func (s *SpotifyService) friends() ([]string, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return nil, ErrNotConnected
	}

	var result struct {
		Friends []struct {
			User struct {
				URI  string `json:"uri"`
				Name string `json:"name"`
			} `json:"user"`
		} `json:"friends"`
	}
	if err := getJSON(ctx, sess, "/presence-view/v1/buddylist", &result); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(result.Friends))
	friends := make([]string, 0, len(result.Friends))
	for _, f := range result.Friends {
		uri := strings.TrimSpace(f.User.URI)
		name := strings.TrimSpace(strings.TrimPrefix(uri, "spotify:user:"))
		if name == "" || name == uri {
			name = strings.TrimSpace(f.User.Name)
		}
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		friends = append(friends, name)
	}
	return friends, nil
}
