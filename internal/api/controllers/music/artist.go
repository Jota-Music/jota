package music_controller

import (
	"jota/server/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func GetArtist(c fiber.Ctx) error {
	uri := c.Params("uri")
	if uri == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "uri is required",
		})
	}

	artist, err := repositories.Use.Music.GetArtist(uri)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(artist)
}

func GetArtistDiscography(c fiber.Ctx) error {
	uri := c.Params("uri")
	if uri == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "uri is required",
		})
	}

	disco, err := repositories.Use.Music.GetArtistDiscography(uri)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(disco)
}

func GetAlbumTracks(c fiber.Ctx) error {
	uri := c.Params("uri")
	if uri == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "uri is required",
		})
	}

	tracks, err := repositories.Use.Music.GetAlbumTracks(uri)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(tracks)
}
