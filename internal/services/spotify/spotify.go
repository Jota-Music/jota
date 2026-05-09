package spotify

const SPOTIFY_API_ENDPOINT = "https://api-partner.spotify.com/pathfinder/v2/query"

type SpotifyService struct {
	token *userToken
}

func NewSpotifyService() *SpotifyService {
	return &SpotifyService{}
}
