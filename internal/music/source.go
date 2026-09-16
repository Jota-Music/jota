package music

import "strings"

type Source interface {
	GetFullPlaylist(id string) (Playlist, error)
	RevalidateFullPlaylist(id string) error
}

type Catalog struct {
	spotify Source
	youtube Source
}

func NewCatalog(spotify, youtube Source) *Catalog {
	return &Catalog{spotify: spotify, youtube: youtube}
}

func (c *Catalog) source(id string) Source {
	if strings.HasPrefix(id, YouTubePrefix) {
		return c.youtube
	}
	return c.spotify
}

func (c *Catalog) GetFullPlaylist(id string) (Playlist, error) {
	return c.source(id).GetFullPlaylist(id)
}

func (c *Catalog) RevalidateFullPlaylist(id string) error {
	return c.source(id).RevalidateFullPlaylist(id)
}
