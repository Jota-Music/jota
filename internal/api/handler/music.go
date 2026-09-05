package handler

import (
	music_controller "jota/server/internal/api/controllers/music"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
)

func Music(app *fiber.App) {
	app.Get("/api/music/playlist/:id", music_controller.GetPlaylist)
	app.Get("/api/music/playlist/full/:id", cache.New(cache.Config{
		Expiration: 12 * time.Hour,
		CacheInvalidator: func(c fiber.Ctx) bool {
			return c.Query("revalidate") == "1"
		},
	}), music_controller.GetFullPlaylist)
	app.Get("/api/music/playlists/:user", cache.New(cache.Config{
		Expiration: 12 * time.Hour,
		CacheInvalidator: func(c fiber.Ctx) bool {
			return c.Query("revalidate") == "1"
		},
	}), music_controller.GetUserPlaylists)
	app.Get("/api/music/song/:id", music_controller.GetSong)
	app.Post("/api/music/link-youtube", music_controller.SetYoutubeId)
	app.Get("/api/music/search/:type/:query", music_controller.Search)
	app.Get("/api/music/artist/:uri", cache.New(cache.Config{
		Expiration: 12 * time.Hour,
		CacheInvalidator: func(c fiber.Ctx) bool {
			return c.Query("revalidate") == "1"
		},
	}), music_controller.GetArtist)
	app.Get("/api/music/artist/:uri/discography", cache.New(cache.Config{
		Expiration: 12 * time.Hour,
		CacheInvalidator: func(c fiber.Ctx) bool {
			return c.Query("revalidate") == "1"
		},
	}), music_controller.GetArtistDiscography)
	app.Get("/api/music/album/:uri", cache.New(cache.Config{
		Expiration: 12 * time.Hour,
		CacheInvalidator: func(c fiber.Ctx) bool {
			return c.Query("revalidate") == "1"
		},
	}), music_controller.GetAlbumTracks)
}
