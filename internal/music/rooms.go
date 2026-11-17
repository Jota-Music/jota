package music

import "strings"

// RoomID names a saved room the way every id in the app is named: the platform
// and kind, then the relay and code the room belongs to. The relay is
// case-insensitive, the code is not.
func RoomID(relayURL string, code string) string {
	return LocalPrefix + "room:" + strings.ToLower(strings.TrimSpace(relayURL)) + "|" + strings.TrimSpace(code)
}

// StoredRoomID maps an id written before the rule (`<relay>|<code>`) to its
// current form. An id that already carries a platform is left alone, so reading
// a store twice changes nothing.
func StoredRoomID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || strings.HasPrefix(id, LocalPrefix) {
		return id
	}
	relay, code, ok := strings.Cut(id, "|")
	if !ok {
		return id
	}
	return RoomID(relay, code)
}
