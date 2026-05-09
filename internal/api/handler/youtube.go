package handler

import (
	music_controller "jota/server/internal/api/controllers/music"

	"github.com/gofiber/fiber/v3"
)

func Youtube(app *fiber.App) {
	app.Get("/api/youtube/audio/:id", music_controller.GetAudio)
	app.Post("/api/youtube/audio/:id", music_controller.SetYoutubeId)
}
