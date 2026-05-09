package user_controller

import (
	"jota/server/internal/services/user"

	"github.com/gofiber/fiber/v3"
)

func SetUserMetadata(c fiber.Ctx) error {
	body := new(user.Metadata)
	if err := c.Bind().Body(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User metadata updated",
	})
}
