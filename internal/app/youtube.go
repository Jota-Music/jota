package app

import (
	"errors"
	"log"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (a *App) ResolveAudio(song music.Song) (music.Audio, error) {
	audio, err := a.YouTube.ResolveAudio(song)
	if err != nil {
		log.Printf("resolve-audio failed id=%q youtubeId=%q name=%q err=%v",
			song.Id, song.YoutubeId, song.Name, err)

		// A sign-in would fix it, so nudge the UI to offer one.
		if errors.Is(err, youtube.ErrLoginRequired) && !youtube.HasAuth() {
			application.Get().Event.Emit("youtube:signin-required", song.Name)
		}
	}
	return audio, err
}

func (a *App) SetYouTubeId(cacheKey string, youtubeId string) error {
	return a.YouTube.SetYoutubeId(cacheKey, youtubeId)
}

func (a *App) SearchYouTube(query string) ([]youtube.Video, error) {
	return a.YouTube.Search(query)
}

func (a *App) SearchYouTubePlaylists(query string) ([]music.PlaylistSummary, error) {
	return a.YouTube.SearchPlaylists(query)
}

func (a *App) GetYouTubePlaylists() ([]music.PlaylistSummary, error) {
	return a.YouTube.Playlists()
}

func (a *App) AddYouTubePlaylist(id string) (music.PlaylistSummary, error) {
	return a.YouTube.AddPlaylist(id)
}

func (a *App) RemoveYouTubePlaylist(id string) error {
	return a.YouTube.RemovePlaylist(id)
}

// SetYouTubeCookies stores the Google account cookies used to sign innertube
// requests. Playing age-restricted videos and avoiding the bot check requires
// it; using an account this way can get it banned by Google.
func (a *App) SetYouTubeCookies(cookies string) error {
	return youtube.SetCookies(cookies)
}

func (a *App) ClearYouTubeCookies() error {
	return youtube.ClearCookies()
}

func (a *App) YouTubeSignedIn() bool {
	return youtube.HasAuth()
}
