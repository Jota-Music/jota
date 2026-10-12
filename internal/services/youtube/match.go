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
// to be harmless, which is why it outweighs the topic bonus. The topic tier must
// stay below the credit bonus by more than a full duration gap, so naming the
// artist in a title can never tie the official upload.
const (
	channelPenalty = -30
	topicPenalty   = -28
	creditBonus    = -20
	lyricPenalty   = 40
)

// How far the uploader's trust survives a length disagreement. Within
// tierFull the tier applies whole; by tierGone the length is so far off that the
// uploader says nothing about it, and the bonus fades in between. Without the
// fade an artist posting a different take — "Yesterday (Take 1)" on The
// Beatles' own channel — outranks the studio master, because the tier is there
// to break ties and not to argue with a quarter of the track missing.
const (
	tierFull = 0.05
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
	"lofi": true, "8d": true, "boosted": true, "muffled": true,
	"demo": true, "tiktok": true,
}

// coverWords announce a re-recording rather than the master: a fan's cover, a
// tribute, a remake. They are held apart from versionWords because the word only
// gives someone away when the uploader is not the artist, so the penalty is
// applied through coverPenalty against the uploader's tier.
var coverWords = map[string]bool{
	"cover": true, "version": true, "tribute": true, "remake": true,
}

// lyricChannels publish karaoke and lyric videos. They restate the song, so
// their running time is the video's and not the track's, which makes them look
// like a perfect duration match.
var lyricChannels = []string{"lyric", "lyrics", "letra", "letras", "lirik", "paroles"}

// accented folds the accented vowels Jota actually meets (Spanish and English
// titles) onto ASCII, so "químicamente" and "quimicamente" compare equal
// without pulling in golang.org/x/text. It also folds the "lo-fi" spellings onto
// one token, so the quality marker matches whatever separator the uploader used.
var accented = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"lo-fi", "lofi", "lo fi", "lofi",
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
	tier := uploadPenalty(v, song)
	return missingTokens(v.Title, song) +
		min(gap*durationWeight, durationPenaltyMax) +
		versionPenalty(v.Title) +
		coverPenalty(v.Title, tier) +
		creditPenalty(v, song, tier) +
		trust(tier, gap)
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

// versionPenalty penalises a title that announces a live cut, a reworked take or
// a degraded re-upload. The count is capped so one word cannot bury a video that
// otherwise matches the track exactly. Covers are handled separately by
// coverPenalty, because naming one only gives the uploader away when it is not
// the artist.
func versionPenalty(title string) float64 {
	var hits int
	for _, token := range tokens(title) {
		if versionWords[token] {
			hits++
		}
	}
	return math.Min(float64(hits)*keywordPenalty, keywordPenaltyMax)
}

// coverPenalty penalises a title that announces a re-recording — a cover, a
// tribute, a remake — but only when the uploader is neither the artist nor their
// Topic channel. The artist's own "(Cover Audio)" single is the master and must
// not be demoted for naming itself honestly; the same word on a stranger's
// channel is the tell that this is not the original recording.
func coverPenalty(title string, tier float64) float64 {
	if tier < 0 {
		return 0
	}
	var hits int
	for _, token := range tokens(title) {
		if coverWords[token] {
			hits++
		}
	}
	return math.Min(float64(hits)*keywordPenalty, keywordPenaltyMax)
}

// creditPenalty rewards an upload whose title names the song's artist, the
// "Artist - Song" convention official and re-uploaded masters use. The channel
// alone is not enough: a label posts under its own name and a fan channel can
// borrow the artist's, so the title is where the performer is actually stated.
// It only fires when the channel was not recognised, so it never stacks on the
// uploader tier, and it is what separates the original from a different artist's
// take on the same title when both match the length.
func creditPenalty(v Video, song music.Song, tier float64) float64 {
	if tier < 0 {
		return 0
	}
	have := map[string]bool{}
	for _, token := range tokens(v.Title) {
		have[token] = true
	}
	for _, artist := range song.Artists {
		want := tokens(artist.Name)
		if len(want) == 0 {
			continue
		}
		named := true
		for _, token := range want {
			if !have[token] {
				named = false
				break
			}
		}
		if named {
			return creditBonus
		}
	}
	return 0
}

// uploadPenalty scores who uploaded the video. The official artist badge, when
// YouTube sends it, settles the channel before anything else: it is the artist's
// own upload even when the title says "lyrics", and without that a same-title
// upload by another artist outranks the master. When there is no badge, the lyric
// words are looked for in the raw title — the tokeniser strips them as noise
// before matching — and checked first and on their own, because a lyric channel
// restates the song and its running time is the video's, not the track's.
func uploadPenalty(v Video, song music.Song) float64 {
	channel := strings.ToLower(strings.TrimSpace(v.Author))
	title := strings.ToLower(v.Title)

	if v.Official {
		return channelPenalty
	}

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
