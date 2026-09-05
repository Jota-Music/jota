package auth_controller

import (
	"jota/server/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func SpotifyStatus(c fiber.Ctx) error {
	svc := repositories.Use.Spotify
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"connected": svc.IsConnected(),
		"user":      svc.Username(),
	})
}

func SpotifyReconnect(c fiber.Ctx) error {
	if err := repositories.Use.Spotify.Reconnect(c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"connected": true,
		"user":      repositories.Use.Spotify.Username(),
	})
}

func SpotifyDisconnect(c fiber.Ctx) error {
	if err := repositories.Use.Spotify.Disconnect(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "disconnected",
	})
}
