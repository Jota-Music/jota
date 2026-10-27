package app

import "github.com/Jota-Music/jota/internal/music"

func (a *App) GetSong(id string) (music.Song, error) {
	return a.Catalog.GetSong(id)
}

func (a *App) GetFullPlaylist(id string) (music.Playlist, error) {
	return a.Catalog.GetFullPlaylist(id)
}

func (a *App) RevalidateFullPlaylist(id string) error {
	return a.Catalog.RevalidateFullPlaylist(id)
}

func (a *App) GetFollowedUsers() ([]string, error) {
	return a.Follows.List()
}

func (a *App) FollowUser(user string) error {
	if err := a.Follows.Follow(user); err != nil {
		return err
	}
	a.syncInBackground()
	return nil
}

func (a *App) UnfollowUser(user string) error {
	if err := a.Follows.Unfollow(user); err != nil {
		return err
	}
	a.syncInBackground()
	return nil
}
