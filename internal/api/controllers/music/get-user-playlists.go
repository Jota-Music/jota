package music_controller

import (
	"jota/server/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func GetUserPlaylists(c fiber.Ctx) error {
	user := c.Params("user")
	if user == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user is required",
		})
	}

	pl, e := repositories.Use.Music.GetUserPlaylists(user)
	if e != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": e.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(pl)
}
