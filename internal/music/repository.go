package music

type Uses interface {
	GetSong(id string) (Song, error)
	GetPlaylist(id string, page int, size int) (Playlist, error)
	GetUserPlaylists(user string) ([]PlaylistSummary, error)
	Search(query string, searchType string) ([]SearchResult, error)
	GetArtist(uri string) (ArtistInfo, error)
	GetArtistDiscography(uri string) (ArtistDiscography, error)
	GetAlbumTracks(uri string) ([]Song, error)
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

func (m *MusicRepository) Search(query string, searchType string) ([]SearchResult, error) {
	return m.use.Search(query, searchType)
}

func (m *MusicRepository) GetArtist(uri string) (ArtistInfo, error) {
	return m.use.GetArtist(uri)
}

func (m *MusicRepository) GetArtistDiscography(uri string) (ArtistDiscography, error) {
	return m.use.GetArtistDiscography(uri)
}

func (m *MusicRepository) GetAlbumTracks(uri string) ([]Song, error) {
	return m.use.GetAlbumTracks(uri)
}

func NewMusicRepository(use Uses) *MusicRepository {
	return &MusicRepository{
		use: use,
	}
}
