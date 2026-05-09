package spotify

import (
	"encoding/json"
	"fmt"
	"jota/server/internal/music"
)

type PersistedQuery struct {
	Version    int    `json:"version"`
	Sha256Hash string `json:"sha256Hash"`
}

type GraphQLExtensions struct {
	PersistedQuery PersistedQuery `json:"persistedQuery"`
}

type QueryParameters struct {
	URI                            string `json:"uri"`
	Limit                          int    `json:"limit"`
	Offset                         int    `json:"offset"`
	IncludeEpisodeContentRatingsV2 bool   `json:"includeEpisodeContentRatingsV2"`
}

type QueryBody struct {
	Variables     QueryParameters   `json:"variables"`
	OperationName string            `json:"operationName"`
	Extensions    GraphQLExtensions `json:"extensions"`
}

type Items struct {
	ItemV2 json.RawMessage `json:"itemV2"`
}

type PlaylistResponse struct {
	Data struct {
		PlaylistV2 struct {
			Content struct {
				Items      []Items `json:"items"`
				PagingInfo struct {
					Limit  int `json:"limit"`
					Offset int `json:"offset"`
				} `json:"pagingInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"content"`
		} `json:"playlistV2"`
	} `json:"data"`
}

type Page struct {
	Size    int  `json:"size"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasNext bool `json:"hasNext"`
}

type Playlist struct {
	Songs []json.RawMessage `json:"songs"`
	Page  Page              `json:"page"`
}

func (s *SpotifyService) GetPlaylist(playlistID string, page int, size int) (music.Playlist, error) {
	if page < 0 {
		page = 0
	}

	if size <= 0 {
		size = 20
	}

	offset := page * size

	body := QueryBody{
		Variables: QueryParameters{
			URI:                            fmt.Sprintf("spotify:playlist:%s", playlistID),
			Limit:                          size,
			Offset:                         offset,
			IncludeEpisodeContentRatingsV2: false,
		},
		OperationName: "fetchPlaylistContents",
		Extensions: GraphQLExtensions{
			PersistedQuery: PersistedQuery{
				Version:    1,
				Sha256Hash: "32b05e92e438438408674f95d0fdad8082865dc32acd55bd97f5113b8579092b",
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return music.Playlist{}, err
	}

	raw, err := s.postGraphQL(jsonBody)
	if err != nil {
		return music.Playlist{}, err
	}

	var parsed PlaylistResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return music.Playlist{}, err
	}

	items := parsed.Data.PlaylistV2.Content.Items

	songs := make([]music.Song, 0, len(items))
	for _, item := range items {
		if len(item.ItemV2) == 0 {
			continue
		}
		song, _, err := ParsePlaylistItemToSong(item.ItemV2)
		if err != nil {
			continue
		}
		songs = append(songs, song)
	}

	total := parsed.Data.PlaylistV2.Content.TotalCount

	hasNext := offset+size < total

	return music.Playlist{
		Songs: songs,
		Page: music.Page{
			Size:    size,
			Offset:  offset,
			Total:   total,
			HasNext: hasNext,
		},
	}, nil
}
