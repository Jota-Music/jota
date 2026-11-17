package music

import (
	"math/big"
	"strings"
)

// A follow is named the way every id is: the platform and kind, then the id the
// source answers.
func ArtistFollow(id string) string {
	return "spotify:artist:" + strings.TrimSpace(id)
}

func ChannelFollow(id string) string {
	return "youtube:channel:" + strings.TrimSpace(strings.TrimPrefix(id, "youtube:"))
}

func UserFollow(handle string) string {
	return "spotify:user:" + strings.TrimSpace(strings.TrimPrefix(handle, "@"))
}

// StoredFollowID maps a follow written before the rule (`artist:<id>`,
// `youtube:<channel>`, a user by the bare handle) to its current form, and an
// artist an older document carried as a 32-hex gid to the base62 id the API
// answers. An id that already carries the rule is answered as it is.
func StoredFollowID(id string) string {
	id = strings.TrimSpace(id)
	switch {
	case id == "":
		return id
	case strings.HasPrefix(id, "youtube:"):
		return ChannelFollow(strings.TrimPrefix(strings.TrimPrefix(id, "youtube:"), "channel:"))
	case strings.HasPrefix(id, "artist:"):
		return ArtistFollow(strings.TrimPrefix(id, "artist:"))
	case strings.HasPrefix(id, "spotify:artist:"):
		return ArtistFollow(publicArtist(strings.TrimPrefix(id, "spotify:artist:")))
	case strings.HasPrefix(id, "user:"):
		return UserFollow(strings.TrimPrefix(id, "user:"))
	case strings.Contains(id, ":"):
		// Another kind of id: only a bare token is a user handle.
		return id
	default:
		return UserFollow(id)
	}
}

// publicArtist turns the 32-hex gid this app's partner services answer with
// into the base62 id Spotify writes in a URI. Anything else is already the id.
func publicArtist(id string) string {
	if len(id) != 32 {
		return id
	}
	for _, char := range id {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return id
		}
	}
	value, ok := new(big.Int).SetString(id, 16)
	if !ok {
		return id
	}
	text := value.Text(62)
	if len(text) > 22 {
		return id
	}
	return strings.Repeat("0", 22-len(text)) + text
}
