package playlists

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

// The full export keeps every playlist in one list, told apart by its source,
// and the metadata of a song lives once in the document's songs map.
func TestExportWithSources(t *testing.T) {
	setup(t)

	s := New(nil)
	created, err := s.Create("Mix")
	if err != nil {
		t.Fatal(err)
	}
	pl, _ := s.load(created.Id)
	pl.Songs = []string{"spotify:s1"}
	pl.Tracks = map[string]music.Song{"spotify:s1": {Id: "spotify:s1", Name: "Local song"}}
	if err := s.save(pl); err != nil {
		t.Fatal(err)
	}

	section, err := s.ExportSection([]Snapshot{{
		Id:     "spotify:p1",
		Source: "spotify",
		Name:   "Liked",
		Owner:  "someone",
		Songs:  []music.Song{{Id: "spotify:s2", Name: "Remote song"}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(section)
	if err != nil {
		t.Fatal(err)
	}

	var document struct {
		Items []struct {
			Id         string    `json:"id"`
			Source     string    `json:"source"`
			Name       string    `json:"name"`
			Owner      string    `json:"owner"`
			ModifiedAt int64     `json:"modifiedAt"`
			Songs      []songRef `json:"songs"`
		} `json:"items"`
		Songs map[string]music.Song `json:"songs"`
	}
	if err := json.Unmarshal([]byte(data), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(document.Items))
	}

	local, remote := document.Items[0], document.Items[1]
	if local.Source != "local" || local.Id != created.Id || local.Name != "Mix" {
		t.Fatalf("local input = %+v", local)
	}
	if len(local.Songs) != 1 || local.Songs[0].Id != "spotify:s1" {
		t.Fatalf("local songs = %+v", local.Songs)
	}
	if remote.Source != "spotify" || remote.Name != "Liked" || remote.Owner != "someone" {
		t.Fatalf("remote input = %+v", remote)
	}
	if len(remote.Songs) != 1 || remote.Songs[0].Id != "spotify:s2" {
		t.Fatalf("remote songs = %+v", remote.Songs)
	}

	if len(document.Songs) != 2 {
		t.Fatalf("songs = %+v", document.Songs)
	}
	if document.Songs["spotify:s1"].Name != "Local song" || document.Songs["spotify:s2"].Name != "Remote song" {
		t.Fatalf("songs = %+v", document.Songs)
	}
	if strings.Contains(string(data), "tracks") || strings.Contains(string(data), "external") {
		t.Fatalf("section keeps a per-playlist or separate section: %s", data)
	}
}

// A playlist the source owns is a snapshot: restoring it must never recreate it
// as a local playlist.
func TestImportSkipsSources(t *testing.T) {
	setup(t)

	s := New(nil)
	section, err := s.ExportSection([]Snapshot{{
		Id:     "youtube:p1",
		Source: "youtube",
		Name:   "Liked",
		Songs:  []music.Song{{Id: "youtube:v1", Name: "Remote song"}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	_, n, err := s.MergeSection(section, Restore, true)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("import applied %d playlists, want 0", n)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("import created playlists: %+v", list)
	}
}

// A pass that could not resolve every playlist keeps the copy the document
// already carried, so a slow source never shrinks the backup.
func TestWithSnapshotsKeepsWhatWasNotResolved(t *testing.T) {
	setup(t)

	s := New(nil)
	section, err := s.ExportSection([]Snapshot{
		{Id: "spotify:p1", Source: "spotify", Name: "Liked"},
		{Id: "youtube:p2", Source: "youtube", Name: "Mixes"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// This pass only answered the first one.
	section = section.WithSnapshots([]Snapshot{{Id: "spotify:p1", Source: "spotify", Name: "Liked again"}})

	names := map[string]string{}
	for _, pl := range section.External {
		names[pl.Id] = pl.Name
	}
	if len(names) != 2 {
		t.Fatalf("external = %+v, want the stored copy of the one that did not answer", section.External)
	}
	if names["spotify:p1"] != "Liked again" {
		t.Fatalf("spotify:p1 = %q, want the fresh copy", names["spotify:p1"])
	}
	if names["youtube:p2"] != "Mixes" {
		t.Fatalf("youtube:p2 = %q, want the stored copy", names["youtube:p2"])
	}

	// A pass that answered nothing leaves the document exactly as it was.
	kept := section.WithSnapshots(nil)
	if len(kept.External) != 2 {
		t.Fatalf("external = %+v, want it untouched", kept.External)
	}
}
