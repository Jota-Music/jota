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
