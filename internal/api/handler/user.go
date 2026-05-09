package handler

import (
	user_controller "jota/server/internal/api/controllers/user"

	"github.com/gofiber/fiber/v3"
)

func User(app *fiber.App) {
	app.Post("/api/user/metadata", user_controller.SetUserMetadata)
}
