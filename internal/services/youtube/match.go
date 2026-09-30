package youtube

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Jota-Music/jota/internal/music"
)

// Penalty weights for ranking search results. A missing title token outweighs
// any duration gap on purpose: a wrong song is a worse failure than a wrong
// version, and a live cut can land within 2% of the studio duration, so
// duration alone can never prove two tracks are the same song.
//
// The title and duration signals are 0 for a perfect match, so the uploader
// takes negative weights to break the ties they leave behind, and the version
// signal takes a positive weight on top of whatever gap it already has.
const (
	missingTokenPenalty = 200
	durationWeight      = 100
	durationPenaltyMax  = 150
	keywordPenalty      = 30
	keywordPenaltyMax   = 90
)

// Uploader tiers, best first. The artist posting their own release is the
// master; a "- Topic" channel restates the record with no editor's cuts and no
// filler; a random channel is nothing in particular; a karaoke or subtitled
// re-upload is described by its video, not the track, so its running time is
// the video's and not the track's and it looks like a perfect length match.
//
// A topic channel sits below the artist's own because a channel named after an
// artist can be impersonated, while a "- Topic" channel is one YouTube opened
// itself. The lyric tier is the only one that has to clear a real duration gap
// to be harmless, which is why it outweighs the topic bonus.
const (
	channelPenalty = -25
	topicPenalty   = -15
	lyricPenalty   = 40
)

// How far the uploader's trust survives a length disagreement. Within
// tierFull the tier applies whole; by tierGone the length is so far off that the
// uploader says nothing about it, and the bonus fades in between. Without the
// fade an artist posting a different take — "Yesterday (Take 1)" on The
// Beatles' own channel — outranks the studio master, because the tier is there
// to break ties and not to argue with a quarter of the track missing.
const (
	tierFull = 0.03
	tierGone = 0.15
)

// topicSuffixes are what YouTube appends to the channel it auto-opens for an
// artist. It localises that suffix — "- Topic" in English, "- Tema" in Spanish
// and Portuguese, "– Thema" in German, "(tema)" in Italian, "– тема" in Russian,
// and in Polish and Arabic it leads with the word instead — so the list cannot
// be complete. It does not have to be: clientContext pins the search language to
// English, and a topic channel that slips past this list is merely unranked
// rather than misfiled.
var topicSuffixes = []string{"- topic", "- tema", "- temas"}

// officialSuffixes are the words YouTube and the labels put after the artist's
// name on a channel the artist really owns. Anything else carrying the artist's
// name is someone else's: "Trueno and MILO J", "Bizarrap and Daddy Yankee" or
// "WOS DS3" all begin with the name and none of them are the artist.
var officialSuffixes = []string{"official", "oficial", "vevo"}

// versionWords mark a video as something other than the studio master: a live
// cut, a reworked or degraded copy. They are matched against the already
// normalised title, so "Live" reads as "live" while "Live It Up" keeps all of
// its words.
//
// "cover" is deliberately absent. Every other word describes a derivative, which
// by definition is not the first release, but an artist who releases a cover as
// a single owns the master of it — and labels name that upload "(Cover Audio)"
// on the artist's own VEVO channel. Penalising it buried Reik's official
// "Ahora Sin Ti (Cover Audio)" under a fan's re-upload for want of one word.
var versionWords = map[string]bool{
	"live": true, "vivo": true, "directo": true, "concert": true,
	"concierto": true, "unplugged": true, "acoustic": true, "acustico": true,
	"karaoke": true, "instrumental": true, "orchestral": true,
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
	gap := durationGap(v.Duration, song.Duration)
	return missingTokens(v.Title, song) +
		min(gap*durationWeight, durationPenaltyMax) +
		versionPenalty(v.Title) +
		trust(uploadPenalty(v, song), gap)
}

// trust scales an uploader tier by how close the video's length is to the
// track's, so the tier only ever breaks ties and never overrides a length that
// says it is a different recording. A gap the two ends cannot measure leaves the
// tier whole, because then length has said nothing either way.
func trust(tier, gap float64) float64 {
	switch {
	case tier >= 0 || gap <= tierFull:
		return tier
	case gap >= tierGone:
		return 0
	}
	return tier * (tierGone - gap) / (tierGone - tierFull)
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

// uploadPenalty scores who uploaded the video. The lyric words are looked for
// in the raw title because the tokeniser strips them as noise before matching,
// and they are checked first and on their own: a lyric video on the artist's own
// channel is still a lyric video.
func uploadPenalty(v Video, song music.Song) float64 {
	channel := strings.ToLower(strings.TrimSpace(v.Author))
	title := strings.ToLower(v.Title)

	for _, word := range lyricChannels {
		if strings.Contains(channel, word) || strings.Contains(title, word) {
			return lyricPenalty
		}
	}

	if channel == "" {
		return 0
	}

	for _, suffix := range topicSuffixes {
		if strings.HasSuffix(channel, suffix) {
			return topicPenalty
		}
	}

	for _, artist := range song.Artists {
		if artist.Name == "" {
			continue
		}
		// The channel the artist owns is the bare name — YouTube tells labels to
		// keep it that way — plus the conventions labels and YouTube do append.
		// A name that merely starts with the artist's is a collaboration channel
		// or a fan's, and must not inherit the artist's trust.
		name := strings.ToLower(artist.Name)
		if channel == name {
			return channelPenalty
		}
		for _, suffix := range officialSuffixes {
			if channel == name+suffix || strings.HasPrefix(channel, name+" "+suffix) {
				return channelPenalty
			}
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
		if !isYear(token) {
			want[token] = true
		}
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

// durationGap is the relative difference between the video's length and the
// track's, which demotes live cuts, extended loops and edited highlights.
func durationGap(candidate, song int) float64 {
	if candidate <= 0 || song <= 0 {
		return 0
	}
	return math.Abs(float64(candidate-song)) / float64(song)
}

// isYear reports a bare four-digit year. Spotify hangs the remaster year onto
// the title ("Blinding Lights - Remastered 2024") and almost no upload repeats
// it, so demanding it would bury the right video one whole song's penalty down
// the list. A year says nothing about which song this is.
func isYear(token string) bool {
	if len(token) != 4 {
		return false
	}
	_, err := strconv.Atoi(token)
	return err == nil
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
