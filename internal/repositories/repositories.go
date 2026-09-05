package repositories

import (
	"jota/server/internal/auth"
	"jota/server/internal/env"
	"jota/server/internal/music"
	"jota/server/internal/services/spotify"
	"jota/server/internal/services/user"
)

type Services struct {
	Music   *music.MusicRepository
	Auth    *auth.AuthRepository
	Spotify *spotify.SpotifyService
}

var Use Services

func Init(cfg env.Config) {
	spotifySvc := spotify.NewSpotifyService(cfg.SpotifyClientID)
	Use = Services{
		Music:   music.NewMusicRepository(spotifySvc),
		Auth:    auth.NewAuthRepository(user.NewUserService()),
		Spotify: spotifySvc,
	}
}
