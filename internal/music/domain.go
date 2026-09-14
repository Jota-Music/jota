package music

const YouTubePrefix = "youtube:"

type Song struct {
	Id        string   `json:"id"`
	Url       string   `json:"url"`
	Name      string   `json:"name"`
	Duration  int      `json:"duration"`
	Share     Share    `json:"share"`
	Album     Album    `json:"album"`
	Artists   []Artist `json:"artists"`
	YoutubeId string   `json:"youtubeId,omitempty"`
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
	Id   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type Page struct {
	Size    int  `json:"size"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasNext bool `json:"hasNext"`
}

type Playlist struct {
	Name  string `json:"name,omitempty"`
	Cover string `json:"cover,omitempty"`
	Songs []Song `json:"songs"`
	Page  Page   `json:"page"`
}

type Audio struct {
	Url      string `json:"url"`
	Duration int    `json:"duration"`
	ExpireAt int64  `json:"expireAt"`
	VideoID  string `json:"videoId"`
}

type PlaylistSummary struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Mosaic   string `json:"mosaic,omitempty"`
	Cover    string `json:"cover,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
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
