package music

type Song struct {
	Id       string   `json:"id"`
	Url      string   `json:"uri"`
	Name     string   `json:"name"`
	Duration int      `json:"duration"`
	Share    Share    `json:"share"`
	Album    Album    `json:"album"`
	Artists  []Artist `json:"artists"`
}

type Share struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

type Album struct {
	Title  string   `json:"title"`
	Url    string   `json:"uri"`
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
	Songs []Song `json:"songs"`
	Page  Page   `json:"page"`
}

// PlaylistSummary represents a lightweight user playlist entry returned by Spotify
type PlaylistSummary struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Mosaic string `json:"mosaic,omitempty"`
	Cover  string `json:"cover,omitempty"`
}

// SearchResult represents a Spotify search result
type SearchResult struct {
	URI        string `json:"uri"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	CoverURL   string `json:"coverUrl,omitempty"`
	Artists    []string `json:"artists,omitempty"`
	OwnerName  string `json:"ownerName,omitempty"`
	TrackCount int32  `json:"trackCount,omitempty"`
}

// ArtistInfo represents an artist with their top tracks
type ArtistInfo struct {
	Name     string `json:"name"`
	URI      string `json:"uri"`
	ImageURL string `json:"imageUrl,omitempty"`
	Tracks   []Song `json:"tracks"`
}

// ArtistDiscography represents an artist with their albums
type ArtistDiscography struct {
	Name   string          `json:"name"`
	URI    string          `json:"uri"`
	Albums []AlbumSummary  `json:"albums"`
}

// AlbumSummary represents a lightweight album entry
type AlbumSummary struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Year   int32  `json:"year"`
	Cover  string `json:"cover,omitempty"`
	Group  string `json:"group"`
}
