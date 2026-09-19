package app

import (
	"testing"
	"time"
)

func TestNeedsDiscordSet(t *testing.T) {
	a := &App{}
	fp := "song\x00artist\x00album\x00art\x00true"

	playing := Media{Title: "song", Artist: "artist", Album: "album", Artwork: "art", Playing: true, Duration: 200, Position: 60}

	cases := []struct {
		name string
		st   *discordState
		m    Media
		fp   string
		want bool
	}{
		{
			"no state yet",
			nil, playing, fp, true,
		},
		{
			"same song, position matches wall clock",
			&discordState{fingerprint: fp, position: 60, setAt: time.Now().Add(-2 * time.Second), timed: true},
			Media{Title: "song", Artist: "artist", Album: "album", Artwork: "art", Playing: true, Duration: 200, Position: 62},
			fp, false,
		},
		{
			"track changed",
			&discordState{fingerprint: fp, position: 60, setAt: time.Now(), timed: true},
			Media{Title: "next", Artist: "artist", Album: "album", Artwork: "art", Playing: true, Duration: 200, Position: 0},
			"next\x00artist\x00album\x00art\x00true", true,
		},
		{
			"duration became known after timeline-free set",
			&discordState{fingerprint: fp, position: 0, setAt: time.Now(), timed: false},
			playing, fp, true,
		},
		{
			"seek 60s forward",
			&discordState{fingerprint: fp, position: 60, setAt: time.Now(), timed: true},
			Media{Title: "song", Artist: "artist", Album: "album", Artwork: "art", Playing: true, Duration: 200, Position: 124},
			fp, true,
		},
		{
			"paused, same track",
			&discordState{fingerprint: fp, position: 60, setAt: time.Now(), timed: true},
			Media{Title: "song", Artist: "artist", Album: "album", Artwork: "art", Playing: false, Duration: 200, Position: 60},
			"song\x00artist\x00album\x00art\x00false", true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := a.needsDiscordSet(tc.st, tc.fp, tc.m); got != tc.want {
				t.Fatalf("needsDiscordSet = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestActivityTimestamps(t *testing.T) {
	a := &App{}
	before := time.Now().UnixMilli()
	act := a.activity(Media{Title: "song", Playing: true, Duration: 100, Position: 25})
	if act.Timestamps == nil {
		t.Fatal("expected timestamps for playing song")
	}
	within := func(name string, got int64, want int64) {
		if got < want-5 || got > want+5 {
			t.Fatalf("%s = %d, want ~%d", name, got, want)
		}
	}
	within("start", act.Timestamps.Start, before-25000)
	within("end", act.Timestamps.End, before+75000)

	act = a.activity(Media{Title: "song", Playing: true, Duration: 0, Position: 0})
	if act.Timestamps != nil {
		t.Fatal("no timestamps when duration unknown")
	}

	act = a.activity(Media{Title: "song", Playing: false, Duration: 100, Position: 50})
	if act.Timestamps != nil {
		t.Fatal("no timestamps when paused")
	}
}
