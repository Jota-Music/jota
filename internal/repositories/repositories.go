package repositories

import (
	"jota/server/internal/auth"
	"jota/server/internal/music"
	"jota/server/internal/services/spotify"
	"jota/server/internal/services/user"
)

type Services struct {
	Music *music.MusicRepository
	Auth  *auth.AuthRepository
}

var Use = Services{
	Music: music.NewMusicRepository(spotify.NewSpotifyService()),
	Auth:  auth.NewAuthRepository(user.NewUserService()),
}
