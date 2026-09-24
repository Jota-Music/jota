package update

// Channel is how this build reached the user. Only AppImage, MacOS, Windows
// and Android can replace themselves; every other channel falls back to a link.
type Channel string

const (
	AppImage Channel = "appimage"
	MacOS    Channel = "macos"
	Windows  Channel = "windows"
	Android  Channel = "android"
	External Channel = "external"
)

type Info struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	Available   bool   `json:"available"`
	Installable bool   `json:"installable"`
	URL         string `json:"url"`
}

// Progress reports download progress. Total is 0 when the server sends no
// Content-Length.
type Progress struct {
	Written int64 `json:"written"`
	Total   int64 `json:"total"`
}
