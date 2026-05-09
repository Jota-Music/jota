package spotify

import (
	"encoding/json"
	"fmt"
	"jota/server/internal/music"
	"strings"
)

type ItemV2 struct {
	Typename string `json:"__typename"`
}

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type CoverArt struct {
	Sources []Image `json:"sources"`
}

type Duration struct {
	TotalMilliseconds int `json:"totalMilliseconds"`
}

type Profile struct {
	Name string `json:"name"`
}

type ArtistItem struct {
	Profile Profile `json:"profile"`
	URI     string  `json:"uri,omitempty"`
}

type Artists struct {
	Items []ArtistItem `json:"items"`
}

type Album struct {
	Name     string   `json:"name"`
	URI      string   `json:"uri"`
	CoverArt CoverArt `json:"coverArt"`
	Artists  Artists  `json:"artists,omitempty"`
}

type Playability struct {
	Playable bool   `json:"playable"`
	Reason   string `json:"reason"`
}

type ContentRating struct {
	Label string `json:"label"`
}

type SharingInfo struct {
	ShareID  string `json:"shareId"`
	ShareURL string `json:"shareUrl"`
}

type Track struct {
	Typename string `json:"__typename"`
	ID       string `json:"id"`
	URI      string `json:"uri"`
	Name     string `json:"name"`

	MediaType string `json:"mediaType"`

	Duration Duration `json:"duration"`

	Playcount   string `json:"playcount"`
	Saved       bool   `json:"saved"`
	TrackNumber int    `json:"trackNumber"`

	ContentRating ContentRating `json:"contentRating"`
	Playability   Playability   `json:"playability"`

	SharingInfo SharingInfo `json:"sharingInfo"`

	AlbumOfTrack Album   `json:"albumOfTrack"`
	FirstArtist  Artists `json:"firstArtist"`
}

type PlaylistTrack struct {
	Typename string `json:"__typename"`
	URI      string `json:"uri"`
	Name     string `json:"name"`

	MediaType string `json:"mediaType"`

	TrackDuration Duration `json:"trackDuration"`

	Playcount   string `json:"playcount"`
	TrackNumber int    `json:"trackNumber"`
	DiscNumber  int    `json:"discNumber"`

	ContentRating ContentRating `json:"contentRating"`
	Playability   Playability   `json:"playability"`

	SharingInfo SharingInfo `json:"sharingInfo"`

	AlbumOfTrack Album   `json:"albumOfTrack"`
	Artists      Artists `json:"artists"`
}

func GetTrackIDFromSpotifyURI(uri string) string {
	const prefix = "spotify:track:"
	if strings.HasPrefix(uri, prefix) {
		return strings.TrimPrefix(uri, prefix)
	}
	return ""
}

const spotifyOpenTrackURL = "https://open.spotify.com/track/"

func trackShareFromParts(trackID string, info SharingInfo) music.Share {
	s := music.Share{
		Id:  strings.TrimSpace(info.ShareID),
		Url: strings.TrimSpace(info.ShareURL),
	}
	if trackID == "" {
		return s
	}
	if s.Url == "" {
		s.Url = spotifyOpenTrackURL + trackID
	}
	if s.Id == "" {
		s.Id = trackID
	}
	return s
}

func MapPlaylistTrackToSong(p PlaylistTrack) (music.Song, error) {
	if p.Typename != "" && p.Typename != "Track" {
		return music.Song{}, fmt.Errorf("invalid typename '%s', expected 'Track'", p.Typename)
	}

	if p.URI == "" {
		return music.Song{}, fmt.Errorf("playlist track missing uri, typename '%s'", p.Typename)
	}

	items := p.Artists.Items
	if len(items) == 0 {
		items = p.AlbumOfTrack.Artists.Items
	}

	artists := make([]music.Artist, 0, len(items))
	for _, it := range items {
		if n := it.Profile.Name; n != "" {
			artists = append(artists, music.Artist{Name: n})
		}
	}

	covers := make([]string, 0, len(p.AlbumOfTrack.CoverArt.Sources))
	for _, src := range p.AlbumOfTrack.CoverArt.Sources {
		if src.URL != "" {
			covers = append(covers, src.URL)
		}
	}

	ms := p.TrackDuration.TotalMilliseconds
	durationSec := ms / 1000
	if ms > 0 && durationSec == 0 {
		durationSec = 1
	}

	id := GetTrackIDFromSpotifyURI(p.URI)

	return music.Song{
		Id:       id,
		Url:      p.URI,
		Name:     p.Name,
		Duration: durationSec,
		Share:    trackShareFromParts(id, p.SharingInfo),
		Album: music.Album{
			Title:  p.AlbumOfTrack.Name,
			Url:    p.AlbumOfTrack.URI,
			Covers: covers,
		},
		Artists: artists,
	}, nil
}

func ParsePlaylistTrackResponse(itemV2 json.RawMessage) (music.Song, error) {
	var wrap struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(itemV2, &wrap); err != nil {
		return music.Song{}, err
	}

	if len(wrap.Data) == 0 {
		return music.Song{}, fmt.Errorf("track response wrapper missing data raw '%s'", clipString(string(itemV2), 400))
	}

	var p PlaylistTrack
	if err := json.Unmarshal(wrap.Data, &p); err != nil {
		return music.Song{}, err
	}

	return MapPlaylistTrackToSong(p)
}

func ExtractTrackFromPlaylistItem(raw json.RawMessage) (Track, ItemV2, error) {
	meta := ItemV2{}

	var env struct {
		Typename string          `json:"__typename"`
		Track    json.RawMessage `json:"track"`
		Item     json.RawMessage `json:"item"`
	}

	if err := json.Unmarshal(raw, &env); err != nil {
		return Track{}, meta, err
	}

	meta.Typename = env.Typename

	payloads := make([]json.RawMessage, 0, 3)
	if len(env.Track) > 0 {
		payloads = append(payloads, env.Track)
	}
	if len(env.Item) > 0 {
		payloads = append(payloads, env.Item)
	}
	payloads = append(payloads, raw)

	var lastErr error

	for _, p := range payloads {
		var t Track
		if err := json.Unmarshal(p, &t); err != nil {
			lastErr = err
			continue
		}
		if t.ID != "" || t.URI != "" {
			return t, meta, nil
		}
	}

	if lastErr != nil {
		return Track{}, meta, fmt.Errorf("failed to extract track for typename '%s' %w", env.Typename, lastErr)
	}

	return Track{}, meta, fmt.Errorf("track missing id and uri for typename '%s'", env.Typename)
}

func ParseTrackToSong(t Track) (music.Song, error) {
	if t.URI == "" {
		return music.Song{}, fmt.Errorf("track missing uri id '%s'", t.ID)
	}

	artists := make([]music.Artist, 0, len(t.FirstArtist.Items))
	for _, it := range t.FirstArtist.Items {
		if n := it.Profile.Name; n != "" {
			artists = append(artists, music.Artist{Name: n})
		}
	}

	covers := make([]string, 0, len(t.AlbumOfTrack.CoverArt.Sources))
	for _, src := range t.AlbumOfTrack.CoverArt.Sources {
		if src.URL != "" {
			covers = append(covers, src.URL)
		}
	}

	ms := t.Duration.TotalMilliseconds
	durationSec := ms / 1000
	if ms > 0 && durationSec == 0 {
		durationSec = 1
	}

	id := t.ID
	if id == "" {
		id = GetTrackIDFromSpotifyURI(t.URI)
	}

	return music.Song{
		Id:       id,
		Url:      t.URI,
		Name:     t.Name,
		Duration: durationSec,
		Share:    trackShareFromParts(id, t.SharingInfo),
		Album: music.Album{
			Title:  t.AlbumOfTrack.Name,
			Url:    t.AlbumOfTrack.URI,
			Covers: covers,
		},
		Artists: artists,
	}, nil
}

func ParsePlaylistItemToSong(itemV2 json.RawMessage) (music.Song, ItemV2, error) {
	var probe struct {
		Typename string `json:"__typename"`
	}

	if err := json.Unmarshal(itemV2, &probe); err != nil {
		return music.Song{}, ItemV2{}, err
	}

	meta := ItemV2{Typename: probe.Typename}

	switch probe.Typename {
	case "TrackResponseWrapper":
		song, err := ParsePlaylistTrackResponse(itemV2)
		return song, meta, err

	default:
		t, m, err := ExtractTrackFromPlaylistItem(itemV2)
		if m.Typename != "" {
			meta.Typename = m.Typename
		}
		if err != nil {
			return music.Song{}, meta, err
		}

		song, err := ParseTrackToSong(t)
		return song, meta, err
	}
}
