package music_controller

import (
	"jota/server/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func GetSong(c fiber.Ctx) error {
	id := c.Params("id")

	song, err := repositories.Use.Music.GetSong(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(song)
}
