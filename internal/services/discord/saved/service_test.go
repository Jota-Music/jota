package saved

import (
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

func youtubePlaylist(id string) music.PlaylistSummary {
	return music.PlaylistSummary{Id: music.YouTubePrefix + id, Name: id}
}

func spotifyPlaylist(id string) music.PlaylistSummary {
	return music.PlaylistSummary{Id: id, Name: id}
}

func TestSavedLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)

	s := New()

	list, err := s.List()
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("empty store = %v", list)
	}

	if err := s.Add(youtubePlaylist("PL1")); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Add(spotifyPlaylist("37i9dQ")); err != nil {
		t.Fatalf("add spotify: %v", err)
	}
	// A repeated save must not duplicate the entry.
	if err := s.Add(youtubePlaylist("PL1")); err != nil {
		t.Fatalf("add duplicate: %v", err)
	}
	if err := s.Add(music.PlaylistSummary{}); err == nil {
		t.Fatal("add without id: want error, got nil")
	}

	list, err = s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %v, want the 2 saved playlists", list)
	}
	if list[0].Id != "youtube:PL1" || list[1].Id != "37i9dQ" {
		t.Fatalf("list = %v, want youtube then spotify", list)
	}

	if err := s.Remove("37i9dQ"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	list, err = s.List()
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(list) != 1 || list[0].Id != "youtube:PL1" {
		t.Fatalf("after remove = %v, want only youtube:PL1", list)
	}
}

func setup(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
	return New()
}

// Saved playlists were stored as a plain summary list before they carried a
// timestamp.
func TestSectionReadsLegacyList(t *testing.T) {
	s := setup(t)
	legacy := []music.PlaylistSummary{{Id: "youtube:PL1", Name: "PL1", Cover: "c"}}
	if err := bucket.SetObject(indexKey, legacy); err != nil {
		t.Fatalf("legacy save: %v", err)
	}

	part, err := s.Section()
	if err != nil {
		t.Fatalf("section: %v", err)
	}
	if len(part.Items) != 1 || part.Items[0].Id != "youtube:PL1" || part.Items[0].At != 0 || part.Items[0].Cover != "c" {
		t.Fatalf("legacy section = %v", part.Items)
	}
}

// Saving after the upgrade rewrites the legacy summary list into the format
// that carries timestamps, keeping the entries it already held.
func TestAddRewritesLegacyList(t *testing.T) {
	s := setup(t)
	legacy := []music.PlaylistSummary{{Id: "youtube:PL1", Name: "PL1", Cover: "c"}}
	if err := bucket.SetObject(indexKey, legacy); err != nil {
		t.Fatalf("legacy save: %v", err)
	}

	if err := s.Add(youtubePlaylist("PL2")); err != nil {
		t.Fatalf("add: %v", err)
	}

	var stored Section
	if err := bucket.GetObject(indexKey, &stored); err != nil {
		t.Fatalf("legacy list was not rewritten: %v", err)
	}
	if len(stored.Items) != 2 {
		t.Fatalf("stored = %v, want both playlists", stored.Items)
	}
	if got := stored.Items[0]; got.Id != "youtube:PL1" || got.Cover != "c" || got.At != 0 {
		t.Fatalf("legacy entry = %v, want it kept untouched", got)
	}
	if got := stored.Items[1]; got.Id != "youtube:PL2" || got.At == 0 {
		t.Fatalf("new entry = %v, want a timestamp", got)
	}
}

func TestMergeSectionHonoursTombstones(t *testing.T) {
	s := setup(t)
	if err := s.Add(youtubePlaylist("PL1")); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Remove("youtube:PL1"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	remote := Section{Items: []Entry{{Id: "youtube:PL1", Name: "PL1", At: 1}, {Id: "youtube:PL2", Name: "PL2", At: 2}}}
	if _, _, err := s.MergeSection(remote, Newer, true); err != nil {
		t.Fatalf("merge: %v", err)
	}
	list, _ := s.List()
	if len(list) != 1 || list[0].Id != "youtube:PL2" {
		t.Fatalf("after merge = %v, want only PL2; a removal must not be undone", list)
	}

	if _, _, err := s.MergeSection(remote, Restore, true); err != nil {
		t.Fatalf("restore: %v", err)
	}
	list, _ = s.List()
	if len(list) != 2 {
		t.Fatalf("after restore = %v, want both playlists", list)
	}
}

// Saving a playlist again after removing it is newer than the tombstone, so it
// wins a background sync instead of waiting for an explicit restore.
func TestMergeSectionNewerSaveBeatsTombstone(t *testing.T) {
	s := setup(t)
	if err := s.Add(spotifyPlaylist("spotify:p1")); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Remove("spotify:p1"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	remote := Section{Items: []Entry{{Id: "spotify:p1", Name: "Kept", At: time.Now().Add(time.Hour).UnixMilli()}}}
	out, _, err := s.MergeSection(remote, Newer, true)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	list, _ := s.List()
	if len(list) != 1 || list[0].Id != "spotify:p1" || list[0].Name != "Kept" {
		t.Fatalf("after merge = %+v, want the newer entry", list)
	}
	if len(out.Removed) != 0 {
		t.Fatalf("tombstone survived a newer save: %v", out.Removed)
	}
}

func TestMergeSectionWithoutApplyLeavesStore(t *testing.T) {
	s := setup(t)
	if err := s.Add(youtubePlaylist("PL1")); err != nil {
		t.Fatalf("add: %v", err)
	}

	remote := Section{Items: []Entry{{Id: "youtube:PL2", Name: "PL2", At: 2}}}
	out, applied, err := s.MergeSection(remote, Newer, false)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if applied != 0 || len(out.Items) != 2 {
		t.Fatalf("applied = %d, union = %v, want 0 and both entries", applied, out.Items)
	}
	list, _ := s.List()
	if len(list) != 1 || list[0].Id != "youtube:PL1" {
		t.Fatalf("store = %v, want PL1 untouched", list)
	}
}
