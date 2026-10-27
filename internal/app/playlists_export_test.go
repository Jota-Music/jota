package app

import (
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

// A playlist saved from a source also shows up in that source's own listing, so
// the export has to keep a single copy of every id and the first one wins.
func TestDedupe(t *testing.T) {
	lists := [][]music.PlaylistSummary{
		{{Id: "spotify:1", Name: "from source"}, {Id: "spotify:2", Name: "only source"}},
		{{Id: "spotify:1", Name: "from saved"}, {Id: "youtube:1", Name: "saved"}},
	}

	got := dedupe(lists)
	if len(got) != 3 {
		t.Fatalf("dedupe returned %d playlists, want 3: %+v", len(got), got)
	}
	want := []string{"spotify:1", "spotify:2", "youtube:1"}
	for i, id := range want {
		if got[i].Id != id {
			t.Fatalf("dedupe[%d] = %q, want %q", i, got[i].Id, id)
		}
	}
	if got[0].Name != "from source" {
		t.Fatalf("kept %q, want the first occurrence", got[0].Name)
	}

	if out := dedupe(nil); len(out) != 0 {
		t.Fatalf("dedupe(nil) = %+v", out)
	}
}
