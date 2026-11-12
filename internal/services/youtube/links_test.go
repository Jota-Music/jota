package youtube

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Jota-Music/jota/internal/cache"
)

// A pin and a pick are stored in different places: the pin in the state, the
// pick in the cache. Writing one never touches the other.
func TestPinAndPickAreStoredSeparately(t *testing.T) {
	const song = "sep-song"
	const pinVideo = "pinVideo001"
	const pickVideo = "pickVideo01"

	svc := NewService()
	if err := svc.SetYoutubeId(song, pinVideo); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	remember(song, pickVideo)

	if got := pinned(song); got != pinVideo {
		t.Fatalf("pin = %q, want %q", got, pinVideo)
	}
	if got := remembered(song); got != pickVideo {
		t.Fatalf("pick = %q, want %q", got, pickVideo)
	}
	// The effective id prefers the pin.
	if got := svc.YoutubeId(song); got != pinVideo {
		t.Fatalf("effective = %q, want pin %q", got, pinVideo)
	}
	// The pick is in its own file, with no extension.
	p, err := cache.RawPath(pickName(song))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("the pick is not in its own file: %v", err)
	}
}

// A pin survives a pick being written for the same song.
func TestAPinSurvivesAPick(t *testing.T) {
	const song = "survive-song"
	const pinVideo = "pinVideo002"
	const pickVideo = "pickVideo02"

	svc := NewService()
	if err := svc.SetYoutubeId(song, pinVideo); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	remember(song, pickVideo)

	if got := pinned(song); got != pinVideo {
		t.Fatalf("pin was lost: %q", got)
	}
}

// Unpinning drops the pin and the pick: otherwise the old pick resurfaces and
// the unpin looks like it did nothing.
func TestUnpinForgetsThePick(t *testing.T) {
	const song = "unpin-song"
	const pinVideo = "pinVideo003"
	const pickVideo = "pickVideo03"

	svc := NewService()
	if err := svc.SetYoutubeId(song, pinVideo); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	remember(song, pickVideo)

	if err := svc.SetYoutubeId(song, ""); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	if got := pinned(song); got != "" {
		t.Fatalf("pin not cleared: %q", got)
	}
	if got := remembered(song); got != "" {
		t.Fatalf("pick survived the unpin: %q", got)
	}
}

// A pick file that does not hold a video id is damage, never a pick.
func TestADamagedPickIsIgnored(t *testing.T) {
	const song = "damaged-song"

	p, err := cache.RawPath(pickName(song))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("not-an-id"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := remembered(song); got != "" {
		t.Fatalf("a damaged pick was returned: %q", got)
	}
}

// The migration moves the picks the old layout kept in the state into the
// cache, so the state only holds pins.
func TestMigrateLinksMovesPicksOutOfTheState(t *testing.T) {
	const song = "migrate-song"
	const video = "dQw4w9WgXcQ"

	seedYoutubeId(song, video)

	MigrateLinks()

	if got := pinned(song); got != "" {
		t.Fatalf("pin not cleared: %q", got)
	}
	if got := remembered(song); got != video {
		t.Fatalf("pick not migrated: %q", got)
	}
}

// remember never writes a pin; SetYoutubeId never writes a pick.
func TestRememberDoesNotWriteAPinAndSetYoutubeIdDoesNotWriteAPick(t *testing.T) {
	const pinSong = "pin-only-song"
	const pickSong = "pick-only-song"
	const video = "dQw4w9WgXcQ"

	// remember must not touch the pin.
	remember(pickSong, video)
	if got := pinned(pickSong); got != "" {
		t.Fatalf("remember wrote a pin: %q", got)
	}

	// SetYoutubeId must not touch the pick.
	svc := NewService()
	if err := svc.SetYoutubeId(pinSong, video); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	if got := remembered(pinSong); got != "" {
		t.Fatalf("SetYoutubeId wrote a pick: %q", got)
	}
}
