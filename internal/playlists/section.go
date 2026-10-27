package playlists

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// exportPlaylist is the portable shape of a playlist. Songs are the references
// in order and Tracks the metadata known for them; the document hoists that
// metadata into one map shared by every playlist. A Source other than "local"
// marks a playlist the source owns: it travels in the file but is never
// restored.
type exportPlaylist struct {
	Id         string                `json:"-"`
	Source     string                `json:"-"`
	Name       string                `json:"-"`
	Owner      string                `json:"-"`
	Songs      []string              `json:"-"`
	Tracks     map[string]music.Song `json:"-"`
	ModifiedAt int64                 `json:"-"`
}

// local reports whether the playlist belongs to this store, which is the only
// kind a restore may touch.
func (p exportPlaylist) local() bool {
	return p.Source == "local"
}

// songRef is one entry of a playlist's songs: the reference only, because the
// song itself is written once in the document's songs map.
type songRef struct {
	Id string `json:"id"`
}

// wireSong is one song as the document writes it. The id is the key in the
// document's songs map, not a field, and everything derivable from it (url,
// share, the album url) is rebuilt on read.
// shortcut: an empty slice and a nil one read back as nil, until a consumer
// needs the difference.
type wireSong struct {
	Name      string      `json:"name,omitempty"`
	Duration  int         `json:"duration,omitempty"`
	YoutubeId string      `json:"youtubeId,omitempty"`
	Broken    bool        `json:"broken,omitempty"`
	Album     *albumRef   `json:"album,omitempty"`
	Artists   []artistRef `json:"artists,omitempty"`
}

// wireAlbum is one album as the document writes it: the id is the key in the
// document's albums map.
type wireAlbum struct {
	Title  string   `json:"title,omitempty"`
	Covers []string `json:"covers,omitempty"`
	Year   int32    `json:"year,omitempty"`
}

// albumRef points at an album in the document's albums map, or carries the
// album inline when it has no id (a YouTube video's album).
type albumRef struct {
	Id string `json:"id,omitempty"`
	wireAlbum
}

// wireArtist is one artist as the document writes it: the id is the key in the
// document's artists map.
type wireArtist struct {
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
}

// artistRef points at an artist in the document's artists map, or carries the
// artist inline when it has no id.
type artistRef struct {
	Id string `json:"id,omitempty"`
	wireArtist
}

// Snapshot is a playlist owned by a source, handed to the export so it travels
// in the document with the local ones.
type Snapshot struct {
	Id     string
	Source string
	Name   string
	Owner  string
	Songs  []music.Song
}

// Section is the playlists part of a backup document. Every playlist is one
// entry in Items, whatever its source, and the section's songs map holds the
// metadata of every reference they list, once each. External holds the playlists
// a source owns: they travel in the file but are never merged into the store.
// Deleted carries the tombstones, so a deletion travels instead of being undone
// by the next device that still has the playlist.
type Section struct {
	Items    []exportPlaylist
	External []exportPlaylist
	Deleted  map[string]int64
}

// Empty reports whether the section carries nothing worth writing.
func (s Section) Empty() bool {
	return len(s.Items) == 0 && len(s.External) == 0 && len(s.Deleted) == 0
}

// Merge modes, for the callers outside this package that decide between a
// background sync and an explicit restore.
const (
	Newer   = mergeNewer
	Restore = mergeRestore
)

// wirePlaylist is a playlist as the document writes it.
type wirePlaylist struct {
	Id         string    `json:"id"`
	Source     string    `json:"source,omitempty"`
	Name       string    `json:"name"`
	Owner      string    `json:"owner,omitempty"`
	ModifiedAt int64     `json:"modifiedAt,omitempty"`
	Songs      []songRef `json:"songs"`
}

func (d Section) MarshalJSON() ([]byte, error) {
	playlists := make([]wirePlaylist, 0, len(d.Items)+len(d.External))
	refs := newRefs()
	for _, group := range [][]exportPlaylist{d.Items, d.External} {
		for _, pl := range group {
			list := make([]songRef, 0, len(pl.Songs))
			for _, ref := range pl.Songs {
				list = append(list, songRef{Id: ref})
				if song, ok := pl.Tracks[ref]; ok {
					refs.song(ref, song)
				}
			}
			playlists = append(playlists, wirePlaylist{pl.Id, pl.Source, pl.Name, pl.Owner, pl.ModifiedAt, list})
		}
	}

	return json.Marshal(struct {
		Items   []wirePlaylist        `json:"items"`
		Songs   map[string]wireSong   `json:"songs,omitempty"`
		Albums  map[string]wireAlbum  `json:"albums,omitempty"`
		Artists map[string]wireArtist `json:"artists,omitempty"`
		Deleted map[string]int64      `json:"deleted,omitempty"`
	}{playlists, refs.songs, refs.albums, refs.artists, d.Deleted})
}

// refs accumulates the document's shared maps while the playlists are written,
// so a song, an album and an artist are stored once each however many playlists
// reference them.
type refs struct {
	songs   map[string]wireSong
	albums  map[string]wireAlbum
	artists map[string]wireArtist
}

func newRefs() *refs {
	return &refs{
		songs:   make(map[string]wireSong),
		albums:  make(map[string]wireAlbum),
		artists: make(map[string]wireArtist),
	}
}

// song records one reference, its album and its artists, and answers the
// reference the playlist keeps.
func (r *refs) song(ref string, song music.Song) string {
	if _, ok := r.songs[ref]; ok {
		return ref
	}
	r.songs[ref] = wireSong{
		Name:      song.Name,
		Duration:  song.Duration,
		YoutubeId: extraVideoId(ref, song.YoutubeId),
		Broken:    song.Broken,
		Album:     r.albumRef(song.Album),
		Artists:   r.artistRefs(song.Artists),
	}
	return ref
}

// albumRef points at the album's id when it has one, so every song of that
// album shares one entry, and writes the album inline otherwise.
func (r *refs) albumRef(album music.Album) *albumRef {
	if album.Id == "" {
		if album.Title == "" && album.Year == 0 && len(album.Covers) == 0 {
			return nil
		}
		return &albumRef{wireAlbum: wireAlbum{Title: album.Title, Covers: album.Covers, Year: album.Year}}
	}
	if _, ok := r.albums[album.Id]; !ok {
		r.albums[album.Id] = wireAlbum{Title: album.Title, Covers: album.Covers, Year: album.Year}
	}
	return &albumRef{Id: album.Id}
}

func (r *refs) artistRefs(artists []music.Artist) []artistRef {
	if len(artists) == 0 {
		return nil
	}
	out := make([]artistRef, 0, len(artists))
	for _, artist := range artists {
		out = append(out, r.artistRef(artist))
	}
	return out
}

// artistRef points at the artist's key when it has an id, so every song of that
// artist shares one entry, and writes the artist inline otherwise.
func (r *refs) artistRef(artist music.Artist) artistRef {
	if artist.Id == "" {
		return artistRef{wireArtist: wireArtist{Name: artist.Name, Source: artist.Source}}
	}
	key := artistKey(artist)
	if _, ok := r.artists[key]; !ok {
		r.artists[key] = wireArtist{Name: artist.Name, Source: artist.Source}
	}
	return artistRef{Id: key}
}

// artistKey names an artist by source and id, so a YouTube channel id can never
// collide with a Spotify one.
func artistKey(artist music.Artist) string {
	if strings.Contains(artist.Id, ":") || artist.Source == "" {
		return artist.Id
	}
	return artist.Source + ":" + artist.Id
}

// extraVideoId writes youtubeId only when the song's id does not already carry
// it, which is the case of a Spotify track linked to a video.
func extraVideoId(ref, youtubeId string) string {
	if youtubeId == videoId(ref) {
		return ""
	}
	return youtubeId
}

// videoId is the YouTube id a reference carries, if any.
func videoId(ref string) string {
	if !strings.HasPrefix(ref, music.YouTubePrefix) {
		return ""
	}
	return strings.TrimPrefix(ref, music.YouTubePrefix)
}

// spotifyId strips the type prefix Spotify ids may carry, so "spotify:track:x",
// "spotify:x" and "x" all answer "x".
func spotifyId(id string) string {
	id = strings.TrimPrefix(id, music.SpotifyPrefix)
	for _, kind := range []string{"track:", "album:", "artist:"} {
		id = strings.TrimPrefix(id, kind)
	}
	return id
}

// songURL is the url a song's id implies, so it never travels in the document.
func songURL(ref string) string {
	if strings.HasPrefix(ref, music.YouTubePrefix) {
		return "https://www.youtube.com/watch?v=" + videoId(ref)
	}
	if strings.HasPrefix(ref, music.SpotifyPrefix) {
		return music.SpotifyPrefix + "track:" + spotifyId(ref)
	}
	return ""
}

// shareURL is the public url of a song. The share id is always the song's own.
func shareURL(ref string) string {
	if strings.HasPrefix(ref, music.YouTubePrefix) {
		return songURL(ref)
	}
	if strings.HasPrefix(ref, music.SpotifyPrefix) {
		return "https://open.spotify.com/track/" + spotifyId(ref)
	}
	return ""
}

// song turns one song entry back into the song the store keeps, rebuilding the
// parts that are derived from the reference.
func (entry wireSong) song(ref string, albums map[string]wireAlbum, artists map[string]wireArtist) music.Song {
	// A reference with nothing else is a bare ref, the shape a deleted track and
	// an unresolved one keep: the id is all that is known, so nothing is rebuilt.
	if entry.Name == "" && entry.Duration == 0 && entry.Album == nil && len(entry.Artists) == 0 && entry.YoutubeId == "" {
		return music.Song{Id: ref, Broken: entry.Broken}
	}

	song := music.Song{
		Id:        ref,
		Url:       songURL(ref),
		Name:      entry.Name,
		Duration:  entry.Duration,
		Share:     music.Share{Id: ref, Url: shareURL(ref)},
		Album:     entry.Album.album(albums),
		Artists:   entry.artists(artists),
		YoutubeId: videoId(ref),
		Broken:    entry.Broken,
	}
	if entry.YoutubeId != "" {
		song.YoutubeId = entry.YoutubeId
	}
	return song
}

func (ref *albumRef) album(albums map[string]wireAlbum) music.Album {
	if ref == nil {
		return music.Album{}
	}
	if ref.Id == "" {
		return ref.wireAlbum.album("")
	}
	known, ok := albums[ref.Id]
	if !ok {
		known = ref.wireAlbum
	}
	return known.album(ref.Id)
}

func (entry wireAlbum) album(id string) music.Album {
	album := music.Album{Id: id, Title: entry.Title, Covers: entry.Covers, Year: entry.Year}
	if strings.HasPrefix(id, music.SpotifyPrefix) {
		album.Url = music.SpotifyPrefix + "album:" + spotifyId(id)
	}
	return album
}

func (entry wireSong) artists(artists map[string]wireArtist) []music.Artist {
	if len(entry.Artists) == 0 {
		return nil
	}
	out := make([]music.Artist, 0, len(entry.Artists))
	for _, ref := range entry.Artists {
		out = append(out, ref.artist(artists))
	}
	return out
}

func (ref artistRef) artist(artists map[string]wireArtist) music.Artist {
	if ref.Id == "" {
		return music.Artist{Name: ref.Name, Source: ref.Source}
	}
	known, ok := artists[ref.Id]
	if !ok {
		known = ref.wireArtist
	}
	id := ref.Id
	if known.Source != "" {
		id = strings.TrimPrefix(id, known.Source+":")
	}
	return music.Artist{Id: id, Name: known.Name, Source: known.Source}
}

// UnmarshalJSON turns the section's shared songs map back into the metadata of
// each playlist that references a song.
func (d *Section) UnmarshalJSON(data []byte) error {
	var wire struct {
		Items   []wirePlaylist        `json:"items"`
		Songs   map[string]wireSong   `json:"songs"`
		Albums  map[string]wireAlbum  `json:"albums"`
		Artists map[string]wireArtist `json:"artists"`
		Deleted map[string]int64      `json:"deleted"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	d.Deleted = wire.Deleted
	d.Items, d.External = nil, nil
	if wire.Items == nil {
		return nil
	}

	d.Items = make([]exportPlaylist, 0, len(wire.Items))
	for _, pl := range wire.Items {
		refs := make([]string, 0, len(pl.Songs))
		var tracks map[string]music.Song
		for _, ref := range pl.Songs {
			if ref.Id == "" {
				continue
			}
			refs = append(refs, ref.Id)
			known, ok := wire.Songs[ref.Id]
			if !ok {
				continue
			}
			if tracks == nil {
				tracks = make(map[string]music.Song, len(pl.Songs))
			}
			tracks[ref.Id] = known.song(ref.Id, wire.Albums, wire.Artists)
		}
		playlist := exportPlaylist{
			Id:         pl.Id,
			Source:     pl.Source,
			Name:       pl.Name,
			Owner:      pl.Owner,
			Songs:      refs,
			Tracks:     tracks,
			ModifiedAt: pl.ModifiedAt,
		}
		if playlist.local() {
			d.Items = append(d.Items, playlist)
			continue
		}
		d.External = append(d.External, playlist)
	}
	return nil
}

// ExportSection serializes every stored playlist into the playlists section of
// a backup document, plus the playlists of the sources, which travel in the same
// list and are skipped when the section is restored.
func (s *Service) ExportSection(external []Snapshot) (Section, error) {
	s.hydrate()

	s.mu.Lock()
	defer s.mu.Unlock()

	ids, err := s.index()
	if err != nil {
		return Section{}, err
	}
	graves, err := s.tombstones()
	if err != nil {
		return Section{}, err
	}

	out := Section{
		Items:    []exportPlaylist{},
		External: snapshots(external),
		Deleted:  graves,
	}
	for _, id := range ids {
		pl, err := s.load(id)
		if err != nil {
			if errors.Is(err, kv.ErrKeyNotFound) {
				continue
			}
			return Section{}, err
		}
		out.Items = append(out.Items, s.portable(pl))
	}
	return out, nil
}

// WithSnapshots replaces the playlists the sources own, leaving the rest of the
// section as it is, so a pass can refresh them without re-reading the store.
func (s Section) WithSnapshots(external []Snapshot) Section {
	// A playlist this pass could not resolve keeps the copy the document already
	// carried: a slow or unavailable source must not shrink the backup.
	if len(external) == 0 {
		return s
	}
	fresh := snapshots(external)
	seen := make(map[string]struct{}, len(fresh))
	for _, pl := range fresh {
		seen[pl.Id] = struct{}{}
	}
	kept := make([]exportPlaylist, 0, len(fresh)+len(s.External))
	kept = append(kept, fresh...)
	for _, pl := range s.External {
		if _, ok := seen[pl.Id]; ok {
			continue
		}
		kept = append(kept, pl)
	}
	s.External = kept
	return s
}

// snapshots turns the source-owned playlists the caller resolved into the
// entries a document carries.
func snapshots(external []Snapshot) []exportPlaylist {
	out := make([]exportPlaylist, 0, len(external))
	for _, ext := range external {
		refs := make([]string, 0, len(ext.Songs))
		tracks := make(map[string]music.Song, len(ext.Songs))
		for _, song := range ext.Songs {
			if song.Id == "" {
				continue
			}
			refs = append(refs, song.Id)
			tracks[song.Id] = song
		}
		out = append(out, exportPlaylist{
			Id:     ext.Id,
			Source: ext.Source,
			Name:   ext.Name,
			Owner:  ext.Owner,
			Songs:  refs,
			Tracks: tracks,
		})
	}
	return out
}

// MergeSection reconciles the store with the playlists section of a document
// and answers the section both sides agree on, plus how many stored playlists
// changed. An explicit restore also undoes local deletions; the playlists of a
// source are a snapshot and are never recreated locally. With apply off the
// union is answered but the store keeps its own playlists.
func (s *Service) MergeSection(part Section, mode mergeMode, apply bool) (Section, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.merge(part, mode, apply)
}
