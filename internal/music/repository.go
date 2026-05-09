package music

type Uses interface {
	GetSong(id string) (Song, error)
	GetPlaylist(id string, page int, size int) (Playlist, error)
	GetUserPlaylists(user string) ([]PlaylistSummary, error)
}

type MusicRepository struct {
	use Uses
}

func (m *MusicRepository) GetSong(id string) (Song, error) {
	return m.use.GetSong(id)
}

func (m *MusicRepository) GetUserPlaylists(user string) ([]PlaylistSummary, error) {
	return m.use.GetUserPlaylists(user)
}

func (m *MusicRepository) GetPlaylist(id string, page int, size int) (Playlist, error) {
	return m.use.GetPlaylist(id, page, size)
}

func NewMusicRepository(use Uses) *MusicRepository {
	return &MusicRepository{
		use: use,
	}
}
