package app

import "github.com/Jota-Music/jota/internal/music"

func (a *App) GetFullPlaylist(id string) (music.Playlist, error) {
	return a.Catalog.GetFullPlaylist(id)
}

func (a *App) RevalidateFullPlaylist(id string) error {
	return a.Catalog.RevalidateFullPlaylist(id)
}

func (a *App) GetFollowedUsers(account string) ([]string, error) {
	return a.Follows.List(account)
}

func (a *App) FollowUser(account string, user string) error {
	return a.Follows.Follow(account, user)
}

func (a *App) UnfollowUser(account string, user string) error {
	return a.Follows.Unfollow(account, user)
}

func (a *App) GetPlaylistOrder(account string) ([]string, error) {
	return a.Order.List(account)
}

func (a *App) SavePlaylistOrder(account string, ids []string) error {
	return a.Order.Save(account, ids)
}
