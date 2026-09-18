package music

import "strings"

type Source interface {
	GetFullPlaylist(id string) (Playlist, error)
	RevalidateFullPlaylist(id string) error
	GetSong(id string) (Song, error)
}

type Catalog struct {
	spotify Source
	youtube Source
	local   Source
}

func NewCatalog(spotify, youtube, local Source) *Catalog {
	return &Catalog{spotify: spotify, youtube: youtube, local: local}
}

func (c *Catalog) source(id string) Source {
	if strings.HasPrefix(id, YouTubePrefix) {
		return c.youtube
	}
	if strings.HasPrefix(id, LocalPrefix) {
		return c.local
	}
	return c.spotify
}

func (c *Catalog) GetFullPlaylist(id string) (Playlist, error) {
	return c.source(id).GetFullPlaylist(id)
}

func (c *Catalog) RevalidateFullPlaylist(id string) error {
	return c.source(id).RevalidateFullPlaylist(id)
}

func (c *Catalog) GetSong(id string) (Song, error) {
	return c.source(id).GetSong(id)
}
