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

func (a *App) GetYouTubeId(cacheKey string) string {
	return a.YouTube.YoutubeId(cacheKey)
}

func (a *App) SearchYouTube(query string) ([]youtube.Video, error) {
	return a.YouTube.Search(query)
}

func (a *App) SearchYouTubePlaylists(query string) ([]music.PlaylistSummary, error) {
	return a.YouTube.SearchPlaylists(query)
}

func (a *App) SearchYouTubeChannels(query string) ([]music.ChannelInfo, error) {
	return a.YouTube.SearchChannels(query)
}

func (a *App) GetYouTubeChannelPlaylists(channel string) ([]music.PlaylistSummary, error) {
	return a.YouTube.GetChannelPlaylists(channel)
}

func (a *App) RevalidateYouTubeChannel(channel string) error {
	return a.YouTube.RevalidateChannel(channel)
}

func (a *App) GetYouTubeChannelVideos(channel string) ([]music.Song, error) {
	return a.YouTube.GetChannelVideos(channel)
}

func (a *App) GetYouTubeChannelInfo(channel string) (music.ChannelInfo, error) {
	return a.YouTube.GetChannelInfo(channel)
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
