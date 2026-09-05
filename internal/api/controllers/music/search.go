package music_controller

import (
	"jota/server/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func Search(c fiber.Ctx) error {
	query := c.Params("query")
	searchType := c.Params("type")

	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "query is required",
		})
	}

	validTypes := map[string]bool{
		"track":    true,
		"album":    true,
		"playlist": true,
		"artist":   true,
		"user":     true,
	}

	if !validTypes[searchType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid search type",
		})
	}

	results, err := repositories.Use.Music.Search(query, searchType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(results)
}