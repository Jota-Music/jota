package handler

import (
	user_controller "jota/server/internal/api/controllers/user"
	"jota/server/internal/session"

	"github.com/gofiber/fiber/v3"
)

func User(app *fiber.App) {
	app.Post("/api/user/metadata", user_controller.SetUserMetadata)

	cookies := app.Group("/api/user/cookies", session.Require())
	cookies.Post("/", user_controller.UploadCookies)
	cookies.Delete("/", user_controller.DeleteCookies)
	cookies.Get("/", user_controller.GetCookiesStatus)
}
