package auth_controller

import (
	"errors"
	"fmt"
	"html"
	"strconv"

	"jota/server/internal/env"
	"jota/server/internal/repositories"
	"jota/server/internal/services/spotify"

	"github.com/gofiber/fiber/v3"
)

func SpotifyStatus(c fiber.Ctx) error {
	svc := repositories.Use.Spotify
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"connected": svc.IsConnected(),
		"user":      svc.Username(),
	})
}

func SpotifyLogin(c fiber.Ctx) error {
	var body struct {
		Origin string `json:"origin"`
	}
	if err := c.Bind().Body(&body); err != nil {
		body.Origin = string(c.Request().Header.Peek("Origin"))
	}

	cfg := env.Load()
	authURL, err := repositories.Use.Spotify.StartInteractiveLogin(body.Origin, cfg.PublicURL, cfg.Port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url": authURL,
	})
}

func SpotifyLoginCallback(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing code")
	}

	origin, err := repositories.Use.Spotify.ResolveLogin(code)
	if err != nil {
		if errors.Is(err, spotify.ErrNoLoginInProgress) {
			return c.Status(fiber.StatusBadRequest).SendString("No login in progress")
		}

		page := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Jota</title>
<style>
  body{margin:0;font-family:system-ui,-apple-system,sans-serif;background:#0c0a09;color:#e4e4e7;display:grid;place-items:center;min-height:100vh}
  .card{max-width:28rem;text-align:center;padding:1.5rem}
  .err{color:#ef4444;font-weight:600}
  a{color:#22c55e;font-weight:600}
</style>
</head>
<body>
  <div class="card">
    <p class="err">Login failed.</p>
    <p>%s</p>
    <p><a href="javascript:history.back()">Go back</a> and try again.</p>
  </div>
</body>
</html>`, html.EscapeString(err.Error()))

		return c.Type("html").Status(fiber.StatusOK).SendString(page)
	}

	target := origin + "/"
	if origin == "" {
		target = "/"
	}

	page := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Jota</title>
<style>
  body{margin:0;font-family:system-ui,-apple-system,sans-serif;background:#0c0a09;color:#e4e4e7;display:grid;place-items:center;min-height:100vh}
  .card{text-align:center}
  a{color:#22c55e;font-weight:600}
</style>
</head>
<body>
  <div class="card">
    <p>Logged in with Spotify.</p>
    <p>Returning to Jota…</p>
    <p><a href="%s">Open the app</a> if nothing happens.</p>
  </div>
  <script>setTimeout(function(){window.location.replace(%s)},800)</script>
</body>
</html>`, html.EscapeString(target), strconv.Quote(target))

	return c.Type("html").Status(fiber.StatusOK).SendString(page)
}

func SpotifyLoginCallbackJSON(c fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind().Body(&body); err != nil || body.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing code",
		})
	}

	if _, err := repositories.Use.Spotify.ResolveLogin(body.Code); err != nil {
		if errors.Is(err, spotify.ErrNoLoginInProgress) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "no login in progress",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"ok":        true,
		"connected": true,
		"user":      repositories.Use.Spotify.Username(),
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
