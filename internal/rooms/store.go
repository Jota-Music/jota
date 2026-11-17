package rooms

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/state"
)

type Store struct {
	mu sync.Mutex
}

func NewStore() *Store {
	return &Store{}
}

// roomID identifies a room by relay and code, named like every other id in the
// app: the relay treats codes as case-sensitive, so only the relay is
// lowercased.
func roomID(relayURL string, code string) string {
	return music.RoomID(relayURL, code)
}

func (s *Store) List() ([]Room, error) {
	rooms, err := state.Read[[]Room](state.Rooms)
	if err != nil {
		return nil, err
	}
	// A room saved before the naming rule carries its bare relay|code key. The
	// id is the door the page uses to join, poll and remove it, so every read
	// answers the current form.
	for i := range rooms {
		rooms[i].Id = roomID(rooms[i].RelayURL, rooms[i].Code)
	}
	return rooms, nil
}

// Save inserts or replaces a room, matched by relay and code. It returns the
// whole list so the caller can refresh without a second read.
func (s *Store) Save(room Room) ([]Room, error) {
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
			return rooms, state.Write(state.Rooms, rooms)
		}
	}
	rooms = append(rooms, room)
	return rooms, state.Write(state.Rooms, rooms)
}

func (s *Store) Remove(id string) ([]Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = music.StoredRoomID(id)
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
	return out, state.Write(state.Rooms, out)
}
