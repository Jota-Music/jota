package rooms

import "encoding/json"

// Room is a room the user saved, with the relay it belongs to so its status
// can be queried later. The relay token and the room password are stored
// alongside so a saved room can be checked and rejoined without retyping them;
// both are kept in the local key-value store in plaintext, like the relay
// settings the app already persists.
type Room struct {
	Id       string `json:"id"`
	Code     string `json:"code"`
	RelayURL string `json:"relayUrl"`
	Token    string `json:"token"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SavedAt  int64  `json:"savedAt"`
}

// RoomStatus is a saved room's live state as reported by the relay. State is the
// relay's cached now-playing playback (a Playback object), absent when the room
// has none or the relay predates the field.
type RoomStatus struct {
	Active  bool            `json:"active"`
	Members int             `json:"members"`
	HasHost bool            `json:"hasHost"`
	Locked  bool            `json:"locked"`
	State   json.RawMessage `json:"state,omitempty"`
}
