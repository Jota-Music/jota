package repositories

import (
	"jota/server/internal/auth"
	"jota/server/internal/music"
	"jota/server/internal/services/spotify"
	"jota/server/internal/services/user"
)

type Services struct {
	Music   *music.MusicRepository
	Auth    *auth.AuthRepository
	Spotify *spotify.SpotifyService
}

var spotifySvc = spotify.NewSpotifyService()

var Use = Services{
	Music:   music.NewMusicRepository(spotifySvc),
	Auth:    auth.NewAuthRepository(user.NewUserService()),
	Spotify: spotifySvc,
}
