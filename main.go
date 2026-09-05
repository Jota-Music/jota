package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"strings"

	"jota/server/internal/api/handler"
	"jota/server/internal/env"
	"jota/server/internal/repositories"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed client/dist/*
var clientFolder embed.FS

func main() {
	enviroment := env.Load()

	// Connect to Spotify BEFORE starting the server (like the reference).
	spotifySvc := repositories.Use.Spotify
	if err := spotifySvc.Connect(context.Background()); err != nil {
		log.Printf("spotify: not connected at startup: %v", err)
	} else {
		log.Printf("spotify: connected as %s", spotifySvc.Username())
	}

	if enviroment.SuperUsername != "" && enviroment.SuperPassword != "" {
		_, err := repositories.Use.Auth.NewUser(enviroment.SuperUsername, enviroment.SuperPassword)
		if err != nil {
			log.Println("super user setup:", err)
		} else {
			log.Println("super user created:", enviroment.SuperUsername)
		}
	}

	app := fiber.New()

	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Use("/ws", func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}

		return fiber.ErrUpgradeRequired
	})

	handler.Auth(app)
	handler.Music(app)
	handler.Youtube(app)
	handler.User(app)
	handler.WS(app)

	distFS, err := fs.Sub(clientFolder, "client/dist")
	if err != nil {
		log.Fatal(err)
	}

	// Explicit caching: hashed assets are immutable; the app shell and the
	// service worker must always be revalidated so updates arrive promptly.
	app.Use(func(c fiber.Ctx) error {
		switch path := c.Path(); {
		case strings.HasPrefix(path, "/assets/"):
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
		case path == "/" || path == "/sw.js" || path == "/registerSW.js" || path == "/index.html" || path == "/manifest.webmanifest":
			c.Set("Cache-Control", "no-cache")
		}
		return c.Next()
	})

	app.Use(static.New("", static.Config{
		FS:     distFS,
		Browse: false,
	}))

	app.Get("/*", func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache")
		c.Path("/index.html")

		return static.New("", static.Config{
			FS: distFS,
		})(c)
	})

	log.Fatal(app.Listen(":" + enviroment.Port))
}
