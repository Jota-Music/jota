package youtube

import (
	"encoding/json"
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

// Reik never published a music video for "Ahora Sin Ti", so YouTube's shelf for
// it is one official upload and a wall of fans. The official one is the band's
// own VEVO channel and the label named it "(Cover Audio)", because Reik released
// someone else's song as a single. Reading "cover" as a version keyword demoted
// the one authoritative upload below a fan re-upload.
func TestRankPrefersTheArtistsOwnCover(t *testing.T) {
	song := music.Song{Name: "Ahora Sin Ti", Duration: 181, Artists: []music.Artist{{Name: "Reik"}}}
	candidates := []Video{
		video("Reik - Ahora Sin Ti (Cover Audio)", 181),
		video("Ahora Sin Ti letra", 180),
		video("Reik - Ahora sin ti", 188),
		video("Reik - Ahora Sin Ti (Video Oficial) 2018 Estreno", 183),
		video("Reik ahora sin ti LYRICS", 184),
		video("Reik - Ahora Sin Ti", 180),
		video("Ahora Sin Ti", 222),
	}

	// The app reads the author from shortBylineText, which shortens reikVEVO to
	// "Reik" — the channel still matches the artist exactly.
	candidates[0].Author = "Reik"
	candidates[1].Author = "REIK"
	candidates[6].Author = "Dorathy Pipes - Topic"

	got := rank(candidates, song)[0]
	if got.Title != "Reik - Ahora Sin Ti (Cover Audio)" {
		t.Fatalf("picked %q by %q", got.Title, got.Author)
	}
}

// A fan's cover shares the title and the exact length of the track, so only the
// uploader gives it away. The artist's own "(Cover Audio)" test above fixes the
// other side: the same word on the artist's own channel stays the master.
func TestRankDemotesAFanCover(t *testing.T) {
	song := music.Song{
		Name:     "Havana",
		Duration: 219,
		Artists:  []music.Artist{{Name: "Camila Cabello"}},
	}
	candidates := []Video{
		{ID: "fan", Title: "Camila Cabello - Havana (cover)", Author: "Random Covers", Duration: 219},
		{ID: "master", Title: "Camila Cabello - Havana", Author: "Camila Cabello", Duration: 219},
	}

	got := first(t, candidates, song)
	if got.ID != "master" {
		t.Fatalf("picked %q by %q, want the artist's own upload", got.Title, got.Author)
	}
}

func TestCoverPenalty(t *testing.T) {
	if got := coverPenalty("Song (Cover)", 0); got != keywordPenalty {
		t.Fatalf("untrusted cover = %v, want %v", got, keywordPenalty)
	}
	// The artist's own cover is the master, not a fan re-upload.
	if got := coverPenalty("Song (Cover Audio)", channelPenalty); got != 0 {
		t.Fatalf("artist cover = %v, want 0", got)
	}
	if got := coverPenalty("Song (Cover) (Tribute)", 0); got != 2*keywordPenalty {
		t.Fatalf("cover + tribute = %v, want %v", got, 2*keywordPenalty)
	}
}

// A "8D audio", a "lo-fi" flip or a "bass boosted" upload restates the song at
// its own length but is not the original recording, so the plain upload has to
// win.
func TestRankDemotesDegradedReuploads(t *testing.T) {
	song := music.Song{Name: "Havana", Duration: 219}
	candidates := []Video{
		video("Camila Cabello - Havana (8D AUDIO)", 219),
		video("Camila Cabello - Havana (slowed + reverb)", 219),
		video("Camila Cabello - Havana (Lo-Fi)", 219),
		video("Camila Cabello - Havana (Bass Boosted)", 219),
		video("Camila Cabello - Havana", 219),
	}

	got := first(t, candidates, song)
	if got.Title != "Camila Cabello - Havana" {
		t.Fatalf("picked %q, want the plain upload", got.Title)
	}
}

// "Te Quise Olvidar" is MDO's hit, co-written and first recorded by Carlos
// Baute. YouTube answers "Te Quise Olvidar MDO" with a shelf where MDO's uploads
// sit on channels that do not carry the band's name, while Baute's own audio is
// four seconds closer to the track's length. Only the title says who is singing,
// so without the credit Baute wins on duration alone.
func TestRankPrefersTheTitledArtist(t *testing.T) {
	song := music.Song{
		Name:     "Te Quise Olvidar",
		Duration: 263,
		Artists:  []music.Artist{{Name: "MDO"}},
	}
	candidates := []Video{
		{ID: "baute", Title: "Te Quise Olvidar - Carlos Baute (Audio HQ)", Author: "Frankjvs", Duration: 259},
		{ID: "mdo", Title: "MDO - Te Quise Olvidar (Audio)", Author: "MDOisback", Duration: 268},
		{ID: "mdo-video", Title: "MDO - Te Quise Olvidar (Video Oficial)", Author: "MdoSME", Duration: 274},
	}

	got := first(t, candidates, song)
	if got.ID != "mdo" {
		t.Fatalf("picked %q by %q, want MDO's own upload", got.Title, got.Author)
	}
}

func TestCreditPenalty(t *testing.T) {
	song := music.Song{Artists: []music.Artist{{Name: "The Weeknd"}}}

	if got := creditPenalty(Video{Title: "The Weeknd - Blinding Lights"}, song, 0); got != creditBonus {
		t.Fatalf("titled artist = %v, want %v", got, creditBonus)
	}
	// The artist's own channel already carries the trust; the credit must not
	// stack on top of it.
	if got := creditPenalty(Video{Title: "The Weeknd - Blinding Lights"}, song, channelPenalty); got != 0 {
		t.Fatalf("trusted channel = %v, want 0", got)
	}
	// A title that names no one, or someone else, earns nothing.
	if got := creditPenalty(Video{Title: "Blinding Lights (Audio)"}, song, 0); got != 0 {
		t.Fatalf("untitled = %v, want 0", got)
	}
	if got := creditPenalty(Video{Title: "Blinding Lights - Dua Lipa"}, song, 0); got != 0 {
		t.Fatalf("other artist = %v, want 0", got)
	}
}

// The Voice performance ranks first for "Fast Car": a different artist at half
// the length. The duration gap has to push it down, and the Black Pumas cover
// that happens to run close to the track's length is caught by its own title.
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
	// Spotify hangs the remaster year onto the title with a dash, outside the
	// brackets, and no upload repeats it: demanding "2024" would put the right
	// upload 200 points below a stranger's.
	remaster := music.Song{Name: "Blinding Lights - Remastered 2024"}
	if got := missingTokens("The Weeknd - Blinding Lights (Official Audio)", remaster); got != 0 {
		t.Fatalf("remaster year = %v, want 0", got)
	}
	// The year is the only thing dropped, not the title itself.
	if got := missingTokens("The Weeknd - Save Your Tears", remaster); got != 2*missingTokenPenalty {
		t.Fatalf("renamed track = %v, want %v", got, 2*missingTokenPenalty)
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

	if got := uploadPenalty(Video{Author: "Queen Official"}, song); got != channelPenalty {
		t.Fatalf("artist channel = %v, want %v", got, channelPenalty)
	}
	if got := uploadPenalty(Video{Author: "QueenVEVO"}, song); got != channelPenalty {
		t.Fatalf("vevo channel = %v, want %v", got, channelPenalty)
	}
	if got := uploadPenalty(Video{Author: "Queen Oficial"}, song); got != channelPenalty {
		t.Fatalf("oficial channel = %v, want %v", got, channelPenalty)
	}
	// Carrying the artist's name is not the same as being the artist: these all
	// turned up under a channel search for the name and none of them is them.
	for _, channel := range []string{"WOS DS3", "Trueno and MILO J", "Rosalia Fandom", "Khea Mohale"} {
		artist := music.Song{Artists: []music.Artist{{Name: strings.Fields(channel)[0]}}}
		if got := uploadPenalty(Video{Author: channel}, artist); got != 0 {
			t.Fatalf("%q = %v, want 0", channel, got)
		}
	}
	// The official artist badge is YouTube's own declaration, so it settles the
	// channel even when the byline cannot be read as the artist's own name.
	if got := uploadPenalty(Video{Author: "OficialLuisMiguel", Official: true}, song); got != channelPenalty {
		t.Fatalf("badged channel = %v, want %v", got, channelPenalty)
	}
	// The badge outranks the lyric words: the artist's own upload titled
	// "(Lyrics Video)" is the master, not a lyric channel's restatement.
	badgedLyric := Video{Title: "Queen - Song (Lyrics Video)", Author: "OficialLuisMiguel", Official: true}
	if got := uploadPenalty(badgedLyric, song); got != channelPenalty {
		t.Fatalf("badged lyric video = %v, want %v", got, channelPenalty)
	}
	// A topic channel is trustworthy but not the artist, so it ranks below them.
	if got := uploadPenalty(Video{Author: "Queen - Topic"}, song); got != topicPenalty {
		t.Fatalf("topic channel = %v, want %v", got, topicPenalty)
	}
	// YouTube localises the suffix, so the Spanish and Portuguese channels are
	// the same tier.
	if got := uploadPenalty(Video{Author: "Queen - Temas"}, song); got != topicPenalty {
		t.Fatalf("temas channel = %v, want %v", got, topicPenalty)
	}
	if got := uploadPenalty(Video{Author: "7clouds"}, song); got != 0 {
		t.Fatalf("unrelated channel = %v, want 0", got)
	}
	// The marker counts in the title too, under a channel with no tell.
	sub := Video{Title: "Queen - Song (Sub. Español + Lyrics)", Author: "sweetblue."}
	if got := uploadPenalty(sub, song); got != lyricPenalty {
		t.Fatalf("lyric upload = %v, want %v", got, lyricPenalty)
	}
	// A lyric video wins over its own uploader: the artist posting one is still
	// a lyric video, and its running time is the video's.
	own := Video{Title: "Queen - Song (Official Lyric Video)", Author: "Queen Official"}
	if got := uploadPenalty(own, song); got != lyricPenalty {
		t.Fatalf("lyric upload by the artist = %v, want %v", got, lyricPenalty)
	}
	// The lyric tier has to be enough to sink an exact length, because that is
	// exactly what a lyric video looks like.
	exact := Video{Title: "Song (Lyrics)", Author: "Letras y Letra", Duration: 219}
	if penalty(exact, music.Song{Name: "Song", Duration: 219}) <= 0 {
		t.Fatal("an exact-length lyric upload must score above a neutral one")
	}
}

// The artist posting a different take is still the artist, but the tier breaks
// ties and cannot argue with a length that is a quarter off: the studio master
// is only listed with the song, while "Take 1" carries the channel's name.
func TestTierDoesNotOutweighALengthGap(t *testing.T) {
	song := music.Song{
		Name:     "Yesterday",
		Duration: 175,
		Artists:  []music.Artist{{Name: "The Beatles"}},
	}
	candidates := []Video{
		{ID: "a", Title: "Yesterday (Remastered 2009)", Author: "The Beatles", Duration: 126},
		{ID: "b", Title: "The Beatles - Yesterday", Author: "The Beatles", Duration: 175},
	}

	got := first(t, candidates, song)
	if got.ID != "b" {
		t.Fatalf("picked %q at the wrong length", got.Title)
	}
	// The tier still has to count when the length agrees, or the whole uploader
	// ladder would be dead weight.
	if trust(channelPenalty, 0.02) != channelPenalty {
		t.Fatal("a near-exact length must keep the tier whole")
	}
	if trust(channelPenalty, 0.5) != 0 {
		t.Fatal("a 50% gap must void the tier")
	}
	if trust(lyricPenalty, 0.5) != lyricPenalty {
		t.Fatal("demoting a lyric upload must never be traded away for length")
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

// The reported bug: YouTube Music answers the query with the artist's own
// channel under a byline ("OficialLuisMiguel") the suffix matcher cannot read,
// while a re-upload names the artist in its title and matches the track's length
// exactly. Only the official artist badge tells them apart.
func TestRankPrefersTheBadgedOfficial(t *testing.T) {
	song := music.Song{
		Name:     "El tiempo que te quede libre",
		Duration: 146,
		Artists:  []music.Artist{{Name: "Luis Miguel"}},
	}
	candidates := []Video{
		{ID: "official", Title: "El tiempo que te quede libre", Author: "OficialLuisMiguel", Duration: 147, Official: true},
		{ID: "reupload", Title: "LUIS MIGUEL - El tiempo que te quede libre", Author: "Crispy Duck", Duration: 146},
	}

	got := first(t, candidates, song)
	if got.ID != "official" {
		t.Fatalf("picked %q by %q, want the official upload", got.Title, got.Author)
	}
}

// A song can be uploaded to the artist's own channel as a lyric video while a
// different artist has a track with the same title. The badge says which channel
// is the artist's, so the official upload must win even though its title says
// "lyrics".
func TestRankPrefersTheOfficialLyricVideo(t *testing.T) {
	song := music.Song{Name: "Reflejos", Duration: 201, Artists: []music.Artist{{Name: "Jere"}}}
	candidates := []Video{
		{ID: "official", Title: "JERE - Reflejos ( Lyrics Vídeo )", Author: "JERE", Duration: 206, Official: true},
		{ID: "impostor", Title: "Reflejos", Author: "Jerix - Topic", Duration: 228},
	}

	got := first(t, candidates, song)
	if got.ID != "official" {
		t.Fatalf("picked %q by %q, want the official upload", got.Title, got.Author)
	}
}

// Without a badge the topic channel is the official signal, and naming the
// artist in a stranger's title must not tie it.
func TestTopicBeatsTitleCredit(t *testing.T) {
	song := music.Song{Name: "Song", Duration: 200, Artists: []music.Artist{{Name: "Artist"}}}
	candidates := []Video{
		{ID: "topic", Title: "Song", Author: "Artist - Topic", Duration: 204},
		{ID: "reupload", Title: "Artist - Song", Author: "Some Channel", Duration: 200},
	}

	got := first(t, candidates, song)
	if got.ID != "topic" {
		t.Fatalf("picked %q by %q, want the topic channel", got.Title, got.Author)
	}
}

func TestExtractVideosReadsOfficialBadge(t *testing.T) {
	var sr searchResponse
	fixture := `{"contents":{"sectionListRenderer":{"contents":[{"itemSectionRenderer":{"contents":[
		{"compactVideoRenderer":{"videoId":"official","title":{"runs":[{"text":"Song"}]},"shortBylineText":{"runs":[{"text":"OficialArtist"}]},
			"ownerBadges":[{"metadataBadgeRenderer":{"icon":{"iconType":"OFFICIAL_ARTIST_BADGE"},"style":"BADGE_STYLE_TYPE_VERIFIED_ARTIST"}}]}},
		{"compactVideoRenderer":{"videoId":"verified","title":{"runs":[{"text":"Song (Lyrics)"}]},"shortBylineText":{"runs":[{"text":"7clouds"}]},
			"ownerBadges":[{"metadataBadgeRenderer":{"icon":{"iconType":"CHECK_CIRCLE_THICK"},"style":"BADGE_STYLE_TYPE_VERIFIED"}}]}}
	]}}]}}}`
	if err := json.Unmarshal([]byte(fixture), &sr); err != nil {
		t.Fatal(err)
	}

	videos := extractVideos(sr)
	if len(videos) != 2 {
		t.Fatalf("extracted %d videos, want 2", len(videos))
	}
	if !videos[0].Official {
		t.Fatal("OFFICIAL_ARTIST_BADGE must mark the video official")
	}
	if videos[1].Official {
		t.Fatal("a plain verified channel must not be marked official")
	}
}
