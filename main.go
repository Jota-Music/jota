package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"time"

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

	// Connect to Spotify BEFORE starting the server (like the reference)
	spotifySvc := repositories.Use.Spotify
	if err := spotifySvc.Connect(context.Background()); err != nil {
		log.Fatal("spotify: failed to connect:", err)
	}
	log.Printf("spotify: connected as %s", spotifySvc.Username())

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

	app.Use(static.New("", static.Config{
		FS:            distFS,
		Browse:        false,
		CacheDuration: 10 * time.Second,
		MaxAge:        3600,
	}))

	app.Get("/*", func(c fiber.Ctx) error {
		c.Path("/index.html")

		return static.New("", static.Config{
			FS: distFS,
		})(c)
	})

	log.Fatal(app.Listen(":" + enviroment.Port))
}
