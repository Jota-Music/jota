package youtube

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/Jota-Music/jota/internal/music"
)

// Penalty weights for ranking search results. A missing title token outweighs
// any duration gap on purpose: a wrong song is a worse failure than a wrong
// version, and a live cut can land within 2% of the studio duration, so
// duration alone can never prove two tracks are the same song.
//
// The title and duration signals are 0 for a perfect match, so the official
// channel takes a negative weight to break the ties they leave behind, and the
// version/lyric signals take positive weights on top of whatever gap they
// already have.
const (
	missingTokenPenalty = 200
	durationWeight      = 100
	durationPenaltyMax  = 150
	lyricChannelPenalty = 20
	keywordPenalty      = 30
	keywordPenaltyMax   = 90
	officialBonus       = -10
)

// versionWords mark a video as something other than the studio master: a live
// cut, a cover, a reworked or degraded copy. They are matched against the
// already normalised title, so "Live" reads as "live" while "Live It Up"
// keeps all of its words.
var versionWords = map[string]bool{
	"live": true, "vivo": true, "directo": true, "concert": true,
	"concierto": true, "unplugged": true, "acoustic": true, "acustico": true,
	"cover": true, "karaoke": true, "instrumental": true, "orchestral": true,
	"remix": true, "edit": true, "bootleg": true, "mashup": true, "medley": true,
	"slowed": true, "sped": true, "nightcore": true, "reverb": true, "reversed": true,
	"loop": true, "extended": true, "reaction": true, "tutorial": true,
	"review": true, "session": true, "rehearsal": true, "compilacion": true,
}

// lyricChannels publish karaoke and lyric videos. They restate the song, so
// their running time is the video's and not the track's, which makes them look
// like a perfect duration match.
var lyricChannels = []string{"lyric", "lyrics", "letra", "letras", "lirik", "paroles"}

// accented folds the accented vowels Jota actually meets (Spanish and English
// titles) onto ASCII, so "químicamente" and "quimicamente" compare equal
// without pulling in golang.org/x/text.
var accented = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
)

// bracketed matches the tails Spotify hangs off a title, as in
// "Song (Remastered 2011)" or "Song [feat. X]", which say nothing about which
// song it is.
var bracketed = regexp.MustCompile(`[({\[][^)}\]]*[)}\]]`)

// noise are the words uploaders and YouTube bolt onto titles without changing
// which song they are.
var noise = map[string]bool{
	"official": true, "video": true, "audio": true,
	"lyric": true, "lyrics": true, "letra": true, "letras": true,
	"hd": true, "hq": true, "4k": true,
	"remaster": true, "remastered": true, "visualizer": true,
	"explicit": true, "clean": true,
}

// stopwords are the English and Spanish filler words. The same list filters the
// song and the candidate, so being generous with it is safe: a word dropped on
// both sides cannot cause a false mismatch. "al" and "the" keep a title like
// "Regreso al sexo" comparable with a candidate that drops the article.
var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "in": true, "on": true,
	"to": true, "and": true, "or": true, "for": true, "at": true, "is": true,
	"el": true, "la": true, "los": true, "las": true, "un": true, "una": true,
	"unos": true, "unas": true, "al": true, "de": true, "del": true, "con": true,
	"para": true, "por": true, "sin": true, "sobre": true, "entre": true,
	"desde": true, "hasta": true, "y": true, "en": true, "lo": true,
	"que": true, "se": true, "su": true, "sus": true, "le": true, "les": true,
}

// rank orders search results by how well each video matches the song, best
// first. It only reorders: the caller still takes the first candidate that
// plays, so a poor match stays a fallback instead of becoming a failure. The
// sort is stable, so videos that score the same keep YouTube's own order.
func rank(videos []Video, song music.Song) []Video {
	sort.SliceStable(videos, func(i, j int) bool {
		return penalty(videos[i], song) < penalty(videos[j], song)
	})
	return videos
}

func penalty(v Video, song music.Song) float64 {
	return missingTokens(v.Title, song) +
		durationGap(v.Duration, song.Duration) +
		versionPenalty(v.Title) +
		uploadPenalty(v, song)
}

// versionPenalty penalises a title that announces a live cut, a cover or a
// reworked copy. The count is capped so one word cannot bury a video that
// otherwise matches the track exactly.
func versionPenalty(title string) float64 {
	var hits int
	for _, token := range tokens(title) {
		if versionWords[token] {
			hits++
		}
	}
	return math.Min(float64(hits)*keywordPenalty, keywordPenaltyMax)
}

// uploadPenalty rewards a video uploaded by the artist themselves and demotes
// the karaoke and subtitled re-uploads, whose title and running time both
// describe the video rather than the record. The lyric words are looked for in
// the raw title because the tokeniser strips them as noise before matching.
func uploadPenalty(v Video, song music.Song) float64 {
	channel := strings.ToLower(strings.TrimSpace(v.Author))
	title := strings.ToLower(v.Title)

	for _, word := range lyricChannels {
		if strings.Contains(channel, word) || strings.Contains(title, word) {
			return lyricChannelPenalty
		}
	}

	if channel == "" {
		return 0
	}
	for _, artist := range song.Artists {
		if artist.Name == "" {
			continue
		}
		// "Queen", "Queen Official" and "Queen - Topic" are all the artist.
		name := strings.ToLower(artist.Name)
		if channel == name || strings.HasPrefix(channel, name+" ") {
			return officialBonus
		}
	}
	return 0
}

// missingTokens penalises every word of the song's title that the candidate's
// title never mentions. This is what demotes a different song by the same
// artist, which the duration check cannot see.
func missingTokens(title string, song music.Song) float64 {
	want := map[string]bool{}
	for _, token := range tokens(bracketed.ReplaceAllString(song.Name, " ")) {
		want[token] = true
	}
	if len(want) == 0 {
		return 0
	}

	have := map[string]bool{}
	for _, token := range tokens(title) {
		have[token] = true
	}

	var missing int
	for token := range want {
		if !have[token] {
			missing++
		}
	}
	return float64(missing) * missingTokenPenalty
}

// durationGap penalises the relative difference between the video's length and
// the track's, which demotes live cuts, extended loops and edited highlights.
func durationGap(candidate, song int) float64 {
	if candidate <= 0 || song <= 0 {
		return 0
	}
	gap := math.Abs(float64(candidate-song)) / float64(song)
	return math.Min(gap*durationWeight, durationPenaltyMax)
}

func tokens(s string) []string {
	fields := strings.FieldsFunc(accented.Replace(strings.ToLower(s)), isSeparator)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if noise[field] || stopwords[field] {
			continue
		}
		out = append(out, field)
	}
	return out
}

func isSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
