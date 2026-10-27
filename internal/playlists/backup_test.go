package playlists

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

// restore merges a section written by hand, the way an explicit import does.
func restore(t *testing.T, s *Service, text string) (int, error) {
	t.Helper()
	var part Section
	if err := json.Unmarshal([]byte(text), &part); err != nil {
		t.Fatal(err)
	}
	_, applied, err := s.MergeSection(part, Restore, true)
	return applied, err
}

func TestPlaylistExportImport(t *testing.T) {
	setup(t)

	s := New(nil)
	a, err := s.Create("A")
	if err != nil {
		t.Fatal(err)
	}
	// Add songs directly by re-creating the playlist record with refs.
	pl, _ := s.load(a.Id)
	pl.Songs = []string{"s1", "s2"}
	pl.ModifiedAt = 10
	if err := s.save(pl); err != nil {
		t.Fatal(err)
	}

	// Newer import should overwrite.
	newer := `{"items":[{"id":"` + a.Id + `","source":"local","name":"A2","songs":[{"id":"s3"}],"modifiedAt":20}]}`
	n, err := restore(t, s, newer)
	if err != nil || n != 1 {
		t.Fatalf("import newer: n=%d err=%v", n, err)
	}
	got, _ := s.load(a.Id)
	if got.Name != "A2" || len(got.Songs) != 1 {
		t.Fatalf("after newer import: %+v", got)
	}

	// Older import must not clobber.
	older := `{"items":[{"id":"` + a.Id + `","source":"local","name":"A3","songs":[{"id":"s4"}],"modifiedAt":5}]}`
	n, err = restore(t, s, older)
	if err != nil || n != 0 {
		t.Fatalf("import older: n=%d err=%v", n, err)
	}
	got, _ = s.load(a.Id)
	if got.Name != "A2" {
		t.Fatalf("older import clobbered: %+v", got)
	}

	// Unknown id creates a new playlist.
	created := `{"items":[{"id":"local:deadbeef","source":"local","name":"B","songs":[],"modifiedAt":1}]}`
	n, err = restore(t, s, created)
	if err != nil || n != 1 {
		t.Fatalf("import new: n=%d err=%v", n, err)
	}
	list, _ := s.List()
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2", len(list))
	}

}

// The document only stores what cannot be derived from the reference, so every
// shape a song can have in the store must come back byte for byte: an album
// track from Spotify, a YouTube video whose album has no id, a Spotify track
// linked to a video, a broken reference, an artist without an id and a
// reference with no metadata at all.
func TestExportImportRebuildsSongsExactly(t *testing.T) {
	setup(t)

	refs := []string{
		"spotify:4cOdK2wGLETKBW3PvgPWqT",
		"youtube:dQw4w9WgXcQ",
		"spotify:7GhIk7Il098yCjg4BQjzvb",
		"spotify:gone",
		"spotify:5ChkMS8OtdzJeqyybCc9R5",
		"youtube:unknown",
	}
	tracks := map[string]music.Song{
		"spotify:4cOdK2wGLETKBW3PvgPWqT": {
			Id:       "spotify:4cOdK2wGLETKBW3PvgPWqT",
			Url:      "spotify:track:4cOdK2wGLETKBW3PvgPWqT",
			Name:     "Never Gonna Give You Up",
			Duration: 213,
			Share: music.Share{
				Id:  "spotify:4cOdK2wGLETKBW3PvgPWqT",
				Url: "https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT",
			},
			Album: music.Album{
				Id:     "spotify:6eT7xZZlY9hNmyf9vGNb2L",
				Title:  "Whenever You Need Somebody",
				Url:    "spotify:album:6eT7xZZlY9hNmyf9vGNb2L",
				Covers: []string{"https://i.scdn.co/image/ab67616d0000b2739365a5c4"},
				Year:   1987,
			},
			Artists: []music.Artist{{Id: "spotify:0gxyHStUsqpMadRV0Di1Qt", Name: "Rick Astley"}},
		},
		"youtube:dQw4w9WgXcQ": {
			Id:        "youtube:dQw4w9WgXcQ",
			Url:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			Name:      "Rick Astley - Never Gonna Give You Up (Official Video)",
			Duration:  213,
			Share:     music.Share{Id: "youtube:dQw4w9WgXcQ", Url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"},
			Album:     music.Album{Title: "YouTube", Covers: []string{"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg"}},
			Artists:   []music.Artist{{Id: "UCuAXFkgsw1L7xaCfnd5JJOw", Name: "Rick Astley", Source: "youtube"}},
			YoutubeId: "dQw4w9WgXcQ",
		},
		"spotify:7GhIk7Il098yCjg4BQjzvb": {
			Id:       "spotify:7GhIk7Il098yCjg4BQjzvb",
			Url:      "spotify:track:7GhIk7Il098yCjg4BQjzvb",
			Name:     "Never Gonna Give You Up",
			Duration: 213,
			Share: music.Share{
				Id:  "spotify:7GhIk7Il098yCjg4BQjzvb",
				Url: "https://open.spotify.com/track/7GhIk7Il098yCjg4BQjzvb",
			},
			Album: music.Album{
				Id:     "spotify:6eT7xZZlY9hNmyf9vGNb2L",
				Title:  "Whenever You Need Somebody",
				Url:    "spotify:album:6eT7xZZlY9hNmyf9vGNb2L",
				Covers: []string{"https://i.scdn.co/image/ab67616d0000b2739365a5c4"},
				Year:   1987,
			},
			Artists:   []music.Artist{{Id: "spotify:0gxyHStUsqpMadRV0Di1Qt", Name: "Rick Astley"}},
			YoutubeId: "dQw4w9WgXcQ",
		},
		// A reference whose source no longer answers keeps only the tombstone.
		"spotify:gone": {Id: "spotify:gone", Broken: true},
		"spotify:5ChkMS8OtdzJeqyybCc9R5": {
			Id:       "spotify:5ChkMS8OtdzJeqyybCc9R5",
			Url:      "spotify:track:5ChkMS8OtdzJeqyybCc9R5",
			Name:     "Compilation track",
			Duration: 180,
			Share: music.Share{
				Id:  "spotify:5ChkMS8OtdzJeqyybCc9R5",
				Url: "https://open.spotify.com/track/5ChkMS8OtdzJeqyybCc9R5",
			},
			Artists: []music.Artist{{Name: "Various Artists"}},
		},
		// A stored reference nobody resolved yet has nothing but its id.
		"youtube:unknown": {Id: "youtube:unknown"},
	}

	s := New(nil)
	summary, err := s.Create("Round trip")
	if err != nil {
		t.Fatal(err)
	}
	pl, err := s.load(summary.Id)
	if err != nil {
		t.Fatal(err)
	}
	pl.Songs = refs
	pl.Tracks = tracks
	pl.ModifiedAt = 10
	if err := s.save(pl); err != nil {
		t.Fatal(err)
	}

	section, err := s.ExportSection(nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(summary.Id); err != nil {
		t.Fatal(err)
	}
	if _, n, err := s.MergeSection(section, Restore, true); err != nil || n != 1 {
		t.Fatalf("import: n=%d err=%v", n, err)
	}
	got, err := s.load(summary.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Songs, refs) {
		t.Fatalf("songs = %v, want %v", got.Songs, refs)
	}
	if !reflect.DeepEqual(got.Tracks, tracks) {
		for ref, want := range tracks {
			if have := got.Tracks[ref]; !reflect.DeepEqual(have, want) {
				t.Errorf("%s:\n got %#v\nwant %#v", ref, have, want)
			}
		}
		t.FailNow()
	}
}

// A backup is only useful if it carries the metadata needed to show a playlist
// without the original sources, so pin that export and import agree.
func TestExportImportKeepsTrackMetadata(t *testing.T) {
	setup(t)

	s := New(nil)
	summary, err := s.Create("Backup")
	if err != nil {
		t.Fatal(err)
	}
	pl, _ := s.load(summary.Id)
	pl.Songs = []string{"spotify:track:a", "youtube:b"}
	pl.Tracks = map[string]music.Song{
		"spotify:track:a": {Id: "spotify:track:a", Name: "Spotify song"},
		"youtube:b":       {Id: "youtube:b", Name: "YouTube song"},
	}
	pl.ModifiedAt = 10
	if err := s.save(pl); err != nil {
		t.Fatal(err)
	}

	section, err := s.ExportSection(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(section.Items) != 1 || len(section.Items[0].Tracks) != 2 {
		t.Fatalf("exported tracks = %+v", section.Items)
	}

	// Restore into an empty store, the way another device would.
	if err := s.Delete(summary.Id); err != nil {
		t.Fatal(err)
	}
	fresh := New(nil)
	if _, n, err := fresh.MergeSection(section, Restore, true); err != nil || n != 1 {
		t.Fatalf("import: n=%d err=%v", n, err)
	}
	got, err := fresh.load(summary.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Songs) != 2 || got.Tracks["youtube:b"].Name != "YouTube song" {
		t.Fatalf("imported playlist = %+v", got)
	}
}

// repeated builds the shape a real library has: several playlists whose songs
// cycle over a handful of albums and artists.
func repeated(playlists, perList int) []exportPlaylist {
	albums := []music.Album{
		{Id: "spotify:album:one", Title: "Whenever You Need Somebody", Url: "spotify:album:one", Covers: []string{"https://i.scdn.co/image/ab67616d0000b273one"}, Year: 1987},
		{Id: "spotify:album:two", Title: "The Game", Url: "spotify:album:two", Covers: []string{"https://i.scdn.co/image/ab67616d0000b273two"}, Year: 1980},
	}
	artists := []music.Artist{
		{Id: "spotify:artist:rick", Name: "Rick Astley"},
		{Id: "spotify:artist:queen", Name: "Queen"},
	}

	out := make([]exportPlaylist, 0, playlists)
	for p := range playlists {
		pl := exportPlaylist{
			Id:         fmt.Sprintf("local:%08d", p),
			Source:     "local",
			Name:       fmt.Sprintf("Mix %d", p),
			ModifiedAt: int64(p),
			Tracks:     make(map[string]music.Song, perList),
		}
		for i := range perList {
			id := fmt.Sprintf("%04d%03d", p, i)
			ref := music.SpotifyPrefix + id
			artist := artists[i%len(artists)]
			pl.Songs = append(pl.Songs, ref)
			pl.Tracks[ref] = music.Song{
				Id:       ref,
				Url:      "spotify:track:" + id,
				Name:     fmt.Sprintf("Song %d of %s", i, artist.Name),
				Duration: 180 + i,
				Share:    music.Share{Id: ref, Url: "https://open.spotify.com/track/" + id},
				Album:    albums[i%len(albums)],
				Artists:  []music.Artist{artist},
			}
		}
		out = append(out, pl)
	}
	return out
}

// A song, an album and an artist are written once each, however many playlists
// and songs reference them: that is what keeps the document small.
func TestExportWritesRepeatedEntitiesOnce(t *testing.T) {
	setup(t)

	s := New(nil)
	seed(t, s, repeated(3, 8))

	section, err := s.ExportSection(nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(section)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Songs   map[string]wireSong   `json:"songs"`
		Albums  map[string]wireAlbum  `json:"albums"`
		Artists map[string]wireArtist `json:"artists"`
	}
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Songs) != 24 {
		t.Fatalf("songs = %d, want 24", len(doc.Songs))
	}
	if len(doc.Albums) != 2 || len(doc.Artists) != 2 {
		t.Fatalf("albums = %d, artists = %d, want one entry per entity", len(doc.Albums), len(doc.Artists))
	}
	// A reference carries the id only: title, covers, year, name and source live
	// in the entry it points at, so no song repeats them.
	for ref, song := range doc.Songs {
		if song.Album == nil || song.Album.Id == "" || song.Album.Title != "" ||
			len(song.Album.Covers) != 0 || song.Album.Year != 0 {
			t.Fatalf("%s repeats its album: %+v", ref, song.Album)
		}
		if len(song.Artists) != 1 || song.Artists[0].Id == "" ||
			song.Artists[0].Name != "" || song.Artists[0].Source != "" {
			t.Fatalf("%s repeats its artist: %+v", ref, song.Artists)
		}
	}
}
