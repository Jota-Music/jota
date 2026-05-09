package room

import "jota/server/internal/music"

type User struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Room struct {
	Id      string       `json:"id"`
	Users   []User       `json:"users"`
	Queue   []music.Song `json:"queue"`
	History []music.Song `json:"history"`
}

func (r *Room) Enqueue(song music.Song) {
	r.Queue = append(r.Queue, song)
}

func (r *Room) Dequeue(song music.Song) {
	for i, s := range r.Queue {
		if s.Id == song.Id {
			r.Queue = append(r.Queue[:i], r.Queue[i+1:]...)
			break
		}
	}
}
