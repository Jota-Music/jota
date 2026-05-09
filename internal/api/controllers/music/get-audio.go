package music_controller

import (
	"jota/server/internal/services/youtube"

	"github.com/gofiber/fiber/v3"
)

func GetAudio(c fiber.Ctx) error {
	search := c.Query("search")
	id := c.Params("id")

	youtubeId, err := youtube.GetSong(id, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	audio, err := youtube.GetAudio(youtubeId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url":      audio.Url,
		"duration": audio.Duration,
		"ttl":      audio.ExpireAt,
		"youtube":  youtubeId,
	})
}
