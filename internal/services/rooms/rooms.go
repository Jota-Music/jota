package rooms

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
)

var bucket = kv.UseBucket("rooms")

const listKey = "saved"

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

type Service struct {
	mu sync.Mutex
}

func New() *Service {
	return &Service{}
}

// roomID identifies a room by relay and code. The relay treats codes as
// case-sensitive, so only the relay is lowercased.
func roomID(relayURL string, code string) string {
	return strings.ToLower(strings.TrimSpace(relayURL)) + "|" + strings.TrimSpace(code)
}

func (s *Service) List() ([]Room, error) {
	var rooms []Room
	if err := bucket.GetObject(listKey, &rooms); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return nil, nil
		}
		return nil, err
	}
	return rooms, nil
}

// Save inserts or replaces a room, matched by relay and code. It returns the
// whole list so the caller can refresh without a second read.
func (s *Service) Save(room Room) ([]Room, error) {
	room.Code = strings.TrimSpace(room.Code)
	room.RelayURL = strings.TrimSpace(room.RelayURL)
	room.Name = strings.TrimSpace(room.Name)
	if room.Code == "" || room.RelayURL == "" {
		return nil, errors.New("room needs a relay and a code")
	}
	if room.Name == "" {
		room.Name = room.Code
	}
	if room.SavedAt == 0 {
		room.SavedAt = time.Now().Unix()
	}
	room.Id = roomID(room.RelayURL, room.Code)

	s.mu.Lock()
	defer s.mu.Unlock()

	rooms, err := s.List()
	if err != nil {
		return nil, err
	}
	for i := range rooms {
		if rooms[i].Id == room.Id {
			rooms[i] = room
			return rooms, bucket.SetObject(listKey, rooms)
		}
	}
	rooms = append(rooms, room)
	return rooms, bucket.SetObject(listKey, rooms)
}

func (s *Service) Remove(id string) ([]Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms, err := s.List()
	if err != nil {
		return nil, err
	}
	out := make([]Room, 0, len(rooms))
	for _, r := range rooms {
		if r.Id != id {
			out = append(out, r)
		}
	}
	return out, bucket.SetObject(listKey, out)
}
