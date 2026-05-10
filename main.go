package main

import (
	"embed"
	"io/fs"
	"log"
	"time"

	"jota/server/internal/api/handler"
	"jota/server/internal/env"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed client/dist/*
var clientFolder embed.FS

func main() {
	enviroment := env.Load()

	app := fiber.New()

	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: enviroment.AllowedOrigins,
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"PATCH",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
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

	app.Get("*", func(c fiber.Ctx) error {
		c.Path("/index.html")

		return static.New("", static.Config{
			FS: distFS,
		})(c)
	})

	log.Fatal(app.Listen(":" + enviroment.Port))
}
