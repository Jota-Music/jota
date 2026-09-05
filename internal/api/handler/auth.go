package handler

import (
	auth_controller "jota/server/internal/api/controllers/auth"
	"jota/server/internal/session"

	"github.com/gofiber/fiber/v3"
)

func Auth(app *fiber.App) {
	app.Post("/api/auth/register", auth_controller.Register)
	app.Post("/api/auth/log-in", auth_controller.LogIn)
	app.Post("/api/auth/log-out", auth_controller.LogOut)
	app.Get("/api/auth/user/:id", auth_controller.GetUser)
	app.Get("/api/auth/me", session.Require(), auth_controller.Me)

	app.Get("/api/spotify/status", auth_controller.SpotifyStatus)
	app.Post("/api/spotify/login", auth_controller.SpotifyLogin)
	app.Get("/api/spotify/login/callback", auth_controller.SpotifyLoginCallback)
	app.Post("/api/spotify/login/callback", auth_controller.SpotifyLoginCallbackJSON)
	app.Get("/login", auth_controller.SpotifyLoginCallback)
	app.Post("/api/spotify/reconnect", auth_controller.SpotifyReconnect)
	app.Post("/api/spotify/disconnect", auth_controller.SpotifyDisconnect)
}
