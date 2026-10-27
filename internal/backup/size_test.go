package backup

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

// id builds a reference that looks like the ones the sources answer, so the size
// and the compression ratio below are not an artefact of padded ids.
func id(prefix string, n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	h := fnv.New64a()
	fmt.Fprintf(h, "%s-%d", prefix, n)
	sum := h.Sum64()
	out := make([]byte, 22)
	for i := range out {
		out[i] = alphabet[sum%uint64(len(alphabet))]
		sum /= uint64(len(alphabet))
	}
	return prefix + string(out)
}

var words = []string{
	"night", "river", "gold", "static", "harbour", "ember", "signal", "paper",
	"ocean", "velvet", "canyon", "silver", "hollow", "fever", "orbit", "dust",
}

// The document is the whole library in a text file, so its size is a feature:
// metadata is indexed (songs, albums, artists) and everything derivable from a
// reference (url, share, album url) is left out. This pins both, and reports
// what compressing it would win, so the decision is not guesswork.
func TestDocumentSize(t *testing.T) {
	const (
		albums  = 30
		artists = 15
		lists   = 5
		perList = 60
	)

	s := empty(t)
	next := 0
	s.Playlists.SetResolver(func(track string) (music.Song, error) {
		n := next
		next++
		album := n % albums
		artist := n % artists
		bare := strings.TrimPrefix(track, music.SpotifyPrefix)
		albumId := id("spotify:album:", album)
		return music.Song{
			Id:       track,
			Url:      "spotify:track:" + bare,
			Name:     fmt.Sprintf("%s %s (%s mix)", words[n%len(words)], words[(n/len(words))%len(words)], words[(n/97)%len(words)]),
			Duration: 180 + n%120,
			Share: music.Share{
				Id:  track,
				Url: "https://open.spotify.com/track/" + bare,
			},
			Album: music.Album{
				Id:     albumId,
				Title:  fmt.Sprintf("%s %s", words[album%len(words)], words[(album*7)%len(words)]),
				Url:    albumId,
				Covers: []string{"https://i.scdn.co/image/ab67616d0000b273" + strings.TrimPrefix(albumId, "spotify:album:")},
				Year:   int32(1990 + album),
			},
			Artists: []music.Artist{{
				Id:   id("spotify:artist:", artist),
				Name: fmt.Sprintf("%s %s", words[artist%len(words)], words[(artist*5)%len(words)]),
			}},
		}, nil
	})

	refs := make([]string, 0, lists*perList)
	for i := range lists * perList {
		refs = append(refs, id("spotify:", i))
	}
	for l := range lists {
		summary, err := s.Playlists.Create(fmt.Sprintf("Mix %d", l))
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		// Playlists overlap on purpose: a song in several playlists must cost one
		// entry, not one per playlist.
		slice := refs[l*perList : (l+1)*perList]
		if err := s.Playlists.AddSongs(summary.Id, slice); err != nil {
			t.Fatalf("add songs: %v", err)
		}
	}

	text, err := s.Export(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	raw := len(text)

	var compressed bytes.Buffer
	zw, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	if _, err := zw.Write([]byte(text)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	tracks := lists * perList
	var shape struct {
		Playlists struct {
			Items []struct {
				Songs []struct {
					Id string `json:"id"`
				} `json:"songs"`
			} `json:"items"`
			Songs   map[string]json.RawMessage `json:"songs"`
			Albums  map[string]json.RawMessage `json:"albums"`
			Artists map[string]json.RawMessage `json:"artists"`
		} `json:"playlists"`
	}
	if err := json.Unmarshal([]byte(text), &shape); err != nil {
		t.Fatalf("shape: %v", err)
	}
	pointers := 0
	for _, pl := range shape.Playlists.Items {
		pointers += len(pl.Songs)
	}
	t.Logf("shape: %d playlists, %d refs, %d songs, %d albums, %d artists",
		len(shape.Playlists.Items), pointers, len(shape.Playlists.Songs), len(shape.Playlists.Albums), len(shape.Playlists.Artists))
	t.Logf("document: %d bytes (%d per track), gzip: %d bytes (%.1f×), %d tracks, %d albums, %d artists",
		raw, raw/tracks, compressed.Len(), float64(raw)/float64(compressed.Len()), tracks, albums, artists)

	// Indexed entities appear once each: the album and the artist of every track
	// point at a single entry.
	if got := bytes.Count([]byte(text), []byte(`"covers"`)); got != albums {
		t.Fatalf("covers written %d times, want once per album (%d)", got, albums)
	}
	if got := bytes.Count([]byte(text), []byte(`"year"`)); got != albums {
		t.Fatalf("year written %d times, want once per album (%d)", got, albums)
	}
	// Every song points at its artist by id; the name lives once in the entry.
	// Names travel for the songs, the artists and the playlists themselves.
	names := bytes.Count([]byte(text), []byte(`"name":`))
	if want := tracks + artists + lists; names != want {
		t.Fatalf("names written %d times, want %d (a song or an artist repeating one)", names, want)
	}

	// Nothing derivable from the reference travels: those fields would multiply
	// the file for no gain, because the reader rebuilds them.
	for _, derived := range []string{`"share"`, `"url"`} {
		if bytes.Contains([]byte(text), []byte(derived)) {
			t.Fatalf("document carries %s, which is rebuilt on read", derived)
		}
	}

	// A generous ceiling: a song costs its fields plus two references, so it fails
	// only if metadata stops being shared (an album or an artist repeated per
	// track lands far above this).
	if perTrack := raw / tracks; perTrack > 600 {
		t.Fatalf("%d bytes per track, want the indexed shape under 600", perTrack)
	}
}
