package music_controller

import (
	"jota/server/internal/music"
	"jota/server/internal/repositories"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v3"
)

const (
	playlistBatchSize    = 50
	parallelRequestLimit = 3
)

func GetPlaylist(c fiber.Ctx) error {
	id := c.Params("id")

	queriedPage := c.Query("page")
	page := 0

	if queriedPage != "" {
		parsedPage, conversionError := strconv.Atoi(queriedPage)
		if conversionError == nil && parsedPage > 0 {
			page = parsedPage - 1
		}
	}

	size, err := strconv.Atoi(c.Query("size", "5"))
	if err != nil || size <= 0 {
		size = 5
	}

	playlist, err := repositories.Use.Music.GetPlaylist(id, page, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(playlist)
}

func GetFullPlaylist(c fiber.Ctx) error {
	id := c.Params("id")

	openingBatch, err := repositories.Use.Music.GetPlaylist(id, 0, playlistBatchSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	totalTracks := openingBatch.Page.Total
	if totalTracks == 0 {
		return c.Status(fiber.StatusOK).JSON(music.Playlist{
			Songs: nil,
			Page: music.Page{
				Size:    0,
				Offset:  0,
				Total:   0,
				HasNext: false,
			},
		})
	}

	batchCount := (totalTracks + playlistBatchSize - 1) / playlistBatchSize
	songsEachBatch := make([][]music.Song, batchCount)
	songsEachBatch[0] = openingBatch.Songs

	if batchCount > 1 {
		if err := fetchLaterPlaylistBatches(id, songsEachBatch, batchCount); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	allSongs := concatenatePlaylistBatches(songsEachBatch, totalTracks)

	return c.Status(fiber.StatusOK).JSON(music.Playlist{
		Songs: allSongs,
		Page: music.Page{
			Size:    len(allSongs),
			Offset:  0,
			Total:   totalTracks,
			HasNext: false,
		},
	})
}

// fetchLaterPlaylistBatches loads batches 1..batchCount-1 in parallel (at most
// parallelRequestLimit requests to Spotify at once). Writes into
// songsEachBatch[1:] without touching index 0.
func fetchLaterPlaylistBatches(playlistID string, songsEachBatch [][]music.Song, batchCount int) error {
	concurrencyLimiter := make(chan struct{}, parallelRequestLimit)
	var waitGroup sync.WaitGroup
	var errorMutex sync.Mutex
	var batchFetchError error

	for batchIndex := 1; batchIndex < batchCount; batchIndex++ {
		waitGroup.Add(1)
		go func(batch int) {
			defer waitGroup.Done()
			concurrencyLimiter <- struct{}{}
			defer func() { <-concurrencyLimiter }()

			batchPlaylist, batchError := repositories.Use.Music.GetPlaylist(playlistID, batch, playlistBatchSize)
			if batchError != nil {
				errorMutex.Lock()
				if batchFetchError == nil {
					batchFetchError = batchError
				}
				errorMutex.Unlock()
				return
			}
			songsEachBatch[batch] = batchPlaylist.Songs
		}(batchIndex)
	}

	waitGroup.Wait()
	return batchFetchError
}

func concatenatePlaylistBatches(songsEachBatch [][]music.Song, totalTracks int) []music.Song {
	allSongs := make([]music.Song, 0, totalTracks)
	for batchIndex := 0; batchIndex < len(songsEachBatch); batchIndex++ {
		allSongs = append(allSongs, songsEachBatch[batchIndex]...)
	}
	if len(allSongs) > totalTracks {
		allSongs = allSongs[:totalTracks]
	}
	return allSongs
}
