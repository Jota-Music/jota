package music

const (
	YouTubePrefix = "youtube:"
	LocalPrefix   = "local:"
)

type Song struct {
	Id        string   `json:"id"`
	Url       string   `json:"url"`
	Name      string   `json:"name"`
	Duration  int      `json:"duration"`
	Share     Share    `json:"share"`
	Album     Album    `json:"album"`
	Artists   []Artist `json:"artists"`
	YoutubeId string   `json:"youtubeId,omitempty"`
	// Broken marks a stored reference that no longer resolves at its source.
	Broken bool `json:"broken,omitempty"`
}

type Share struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

type Album struct {
	Id     string   `json:"id"`
	Title  string   `json:"title"`
	Url    string   `json:"url"`
	Covers []string `json:"covers"`
}

type Artist struct {
	Id     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
}

type Playlist struct {
	Name    string `json:"name,omitempty"`
	Cover   string `json:"cover,omitempty"`
	Owner   string `json:"owner,omitempty"`
	OwnerId string `json:"ownerId,omitempty"`
	Songs   []Song `json:"songs"`
}

type Audio struct {
	Url        string `json:"url"`
	Duration   int    `json:"duration"`
	ExpireAt   int64  `json:"expireAt"`
	VideoID    string `json:"videoId"`
	ClientName string `json:"clientName,omitempty"`
	// LoudnessDb is the source loudness YouTube measured (dB relative to its
	// reference level); nil when the video has not been measured.
	LoudnessDb *float64 `json:"loudnessDb,omitempty"`
}

type PlaylistSummary struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Cover    string   `json:"cover,omitempty"`
	Covers   []string `json:"covers,omitempty"`
	Subtitle string   `json:"subtitle,omitempty"`
	Owner    string   `json:"owner,omitempty"`
}

type SearchResult struct {
	URI        string   `json:"uri"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	CoverURL   string   `json:"coverUrl,omitempty"`
	Artists    []string `json:"artists,omitempty"`
	OwnerName  string   `json:"ownerName,omitempty"`
	TrackCount int32    `json:"trackCount,omitempty"`
}

type UserProfile struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

type ChannelInfo struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type Follow struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl,omitempty"`
	Kind     string `json:"kind"`
}

type ArtistInfo struct {
	Name     string `json:"name"`
	URI      string `json:"uri"`
	ImageURL string `json:"imageUrl,omitempty"`
	Tracks   []Song `json:"tracks"`
}

type ArtistDiscography struct {
	Name   string         `json:"name"`
	URI    string         `json:"uri"`
	Albums []AlbumSummary `json:"albums"`
}

type AlbumSummary struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Year  int32  `json:"year"`
	Cover string `json:"cover,omitempty"`
	Group string `json:"group"`
}
