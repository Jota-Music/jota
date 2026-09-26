package youtube

import (
	"strings"
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

// The candidates below are verbatim from the music.youtube.com video shelf for
// the query the app builds, in the order YouTube returned them.

func video(title string, seconds int) Video {
	return Video{ID: "id", Title: title, Duration: seconds}
}

func first(t *testing.T, videos []Video, song music.Song) Video {
	t.Helper()
	ranked := rank(videos, song)
	if len(ranked) == 0 {
		t.Fatal("rank returned nothing")
	}
	return ranked[0]
}

func assertNotFirst(t *testing.T, videos []Video, song music.Song, title string) {
	t.Helper()
	for i, v := range rank(videos, song) {
		if v.Title == title {
			if i == 0 {
				t.Fatalf("%q ranked first, want it demoted", title)
			}
			return
		}
	}
	t.Fatalf("%q not in results", title)
}

// Havana is 3:39. The top hit is the 6:43 music video with the cameo, and the
// live Grammy cut sits third; the studio audio is the exact length.
func TestRankPrefersTheStudioLength(t *testing.T) {
	song := music.Song{Name: "Havana", Duration: 219}
	candidates := []Video{
		video("Camila Cabello - Havana (Official Video) ft. Young Thug", 403),
		video("Camila Cabello - Havana (Audio) ft. Young Thug", 219),
		video("Camila Cabello - Havana (Lyrics) ft. Young Thug", 213),
		video("Camila Cabello - Havana (Vertical Video) ft. Young Thug", 217),
		video("Camila Cabello - Havana (Young Thug only)", 101),
		video("Camila Cabello - Havana (Without Young Thug)", 217),
		video("Camila Cabello - Havana (Live - Audio)", 251),
		video("Havana (LIVE at the 61st GRAMMYs)", 292),
	}

	got := first(t, candidates, song)
	if !strings.Contains(got.Title, "(Audio)") {
		t.Fatalf("picked %q, want the 3:39 studio audio", got.Title)
	}
}

// The Voice performance ranks first for "Fast Car": a different artist at half
// the length. The duration gap has to push it down. Note the remaining v1 gap:
// Black Pumas is a cover by another band that happens to run 282s, so length
// alone still prefers it over the 267s official upload. Catching that needs the
// channel and version-keyword signals planned for v2.
func TestRankRejectsAMismatchedLength(t *testing.T) {
	song := music.Song{Name: "Fast Car", Duration: 296}
	candidates := []Video{
		video("Ceri Hall Brady - Fast Car | The Voice 2022 (Germany) | All Around The Voice", 130),
		video("Black Pumas - Fast Car (Tracy Chapman Cover) (Live on The Tonight Show)", 282),
		video("Jonas Blue - Fast Car ft. Dakota (Official Video)", 216),
		video("LethalBizzle - Fast Car", 182),
		video("Tracy Chapman - Fast Car (Official Music Video)", 267),
		video("Tracy Chapman - Fast Car (Live)", 342),
	}

	assertNotFirst(t, candidates, song, "Ceri Hall Brady - Fast Car | The Voice 2022 (Germany) | All Around The Voice")
}

// Searching SULTANESSA returns only other Freya Ridings songs. The correct track
// is not in the results, so ranking cannot fix the pick, but it must not hand
// back a confident same-artist mismatch while a title shares no word at all.
func TestRankDemotesADifferentSong(t *testing.T) {
	song := music.Song{Name: "SULTANESSA", Duration: 178}
	candidates := []Video{
		video("Freya Ridings - Castles", 209),
		video("Freya Ridings - Castles (Live At The Barbican)", 199),
		video("Freya Ridings - Castles (Official Lyric Video)", 212),
		video("Freya Ridings - Undefeated (Official Lyric Video)", 263),
		video("Freya Ridings - Wicker Woman (Official Video)", 192),
		video("Freya Ridings - Euphoria (Official Lyric Video)", 172),
	}

	assertNotFirst(t, candidates, song, "Freya Ridings - Castles")
}

func TestDurationGapNeedsBothLengths(t *testing.T) {
	// An unknown length on either side leaves the duration out of the score
	// rather than treating the video as a mismatch.
	if got := durationGap(0, 219); got != 0 {
		t.Fatalf("candidate without length = %v, want 0", got)
	}
	if got := durationGap(403, 0); got != 0 {
		t.Fatalf("song without length = %v, want 0", got)
	}
	if got := durationGap(219, 219); got != 0 {
		t.Fatalf("exact length = %v, want 0", got)
	}
	if got := durationGap(403, 219); got <= durationGap(219, 219) {
		t.Fatalf("a 6:43 video must score worse than the exact length")
	}
}

func TestMissingTokens(t *testing.T) {
	song := music.Song{Name: "Regreso al sexo químicamente puro (Remastered 2011)"}

	if got := missingTokens("Ilegales - Regreso al sexo quimicamente puro", song); got != 0 {
		t.Fatalf("accented match = %v, want 0", got)
	}
	if got := missingTokens("Ilegales - Regreso (Official Video)", song); got != 3*missingTokenPenalty {
		t.Fatalf("three words missing = %v, want %v", got, 3*missingTokenPenalty)
	}
	// A title made only of stopwords and noise leaves nothing to compare, so it
	// must not penalise every candidate equally.
	if got := missingTokens("The Official Video", music.Song{Name: "The (Official)"}); got != 0 {
		t.Fatalf("empty token set = %v, want 0", got)
	}
}

// A subtitled lyric re-upload runs within a second of the studio length, so
// only the upload itself gives it away: the title carries the marker and the
// channel is a random name. The official remaster has to win.
func TestRankPrefersTheOfficialUpload(t *testing.T) {
	song := music.Song{
		Name:     "Bohemian Rhapsody",
		Duration: 354,
		Artists:  []music.Artist{{Name: "Queen"}},
	}
	candidates := []Video{
		{ID: "a", Title: "Queen - Bohemian Rhapsody (Sub. Español + Lyrics)", Author: "sweetblue.", Duration: 355},
		{ID: "b", Title: "Queen – Bohemian Rhapsody (Official Video Remastered HD)", Author: "Queen Official", Duration: 360},
		{ID: "c", Title: "Queen - Bohemian Rhapsody (Live Aid 1985)", Author: "Live Aid and Queen Official", Duration: 165},
		{ID: "d", Title: "Queen - Bohemian Rhapsody (Karaoke Version)", Author: "Sing King", Duration: 375},
		{ID: "e", Title: "Benson Boone with Special Guest Brian May - Bohemian Rhapsody", Author: "Benson Boone", Duration: 374},
	}

	got := first(t, candidates, song)
	if got.Author != "Queen Official" {
		t.Fatalf("picked %q by %q, want the official upload", got.Title, got.Author)
	}
}

func TestVersionPenalty(t *testing.T) {
	if got := versionPenalty("INXS - Kick (Live at Wembley Stadium, 1991)"); got != keywordPenalty {
		t.Fatalf("one version word = %v, want %v", got, keywordPenalty)
	}
	if got := versionPenalty("Song (Slowed + Reverb + Nightcore)"); got != keywordPenaltyMax {
		t.Fatalf("three version words = %v, want the cap %v", got, keywordPenaltyMax)
	}
	// Known limitation: the match is on whole words anywhere in the title, so a
	// track with "live" in its own name ("Live It Up") is treated as a live
	// recording. The penalty is one duration gap wide, so it only ever decides
	// between candidates that were already close.
	if got := versionPenalty("Major Lazer - Live It Up"); got != keywordPenalty {
		t.Fatalf("live in the song name = %v, want %v", got, keywordPenalty)
	}
}

func TestUploadPenalty(t *testing.T) {
	song := music.Song{Artists: []music.Artist{{Name: "Queen"}}}

	if got := uploadPenalty(Video{Author: "Queen Official"}, song); got != officialBonus {
		t.Fatalf("artist channel = %v, want %v", got, officialBonus)
	}
	if got := uploadPenalty(Video{Author: "Queen - Topic"}, song); got != officialBonus {
		t.Fatalf("topic channel = %v, want %v", got, officialBonus)
	}
	if got := uploadPenalty(Video{Author: "7clouds"}, song); got != 0 {
		t.Fatalf("unrelated channel = %v, want 0", got)
	}
	// The marker counts in the title too, under a channel with no tell.
	sub := Video{Title: "Queen - Song (Sub. Español + Lyrics)", Author: "sweetblue."}
	if got := uploadPenalty(sub, song); got != lyricChannelPenalty {
		t.Fatalf("lyric upload = %v, want %v", got, lyricChannelPenalty)
	}
}

func TestTokens(t *testing.T) {
	got := tokens("The Weeknd - Blinding Lights (Official Music Video) [4K]")
	want := []string{"weeknd", "blinding", "lights", "music"}

	if len(got) != len(want) {
		t.Fatalf("tokens = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tokens = %v, want %v", got, want)
		}
	}
}
