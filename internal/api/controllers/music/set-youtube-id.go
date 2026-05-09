package music_controller

import (
	"jota/server/internal/services/youtube"

	"github.com/gofiber/fiber/v3"
)

func SetYoutubeId(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		// allow id in query or body as fallback
		id = c.Query("id")
	}

	// try parse body first
	var body struct {
		Id        string `json:"id"`
		YoutubeId string `json:"youtubeId"`
	}
	youtubeId := ""
	if err := c.Bind().Body(&body); err == nil {
		if body.Id != "" && id == "" {
			id = body.Id
		}
		youtubeId = body.YoutubeId
	}
	if youtubeId == "" {
		youtubeId = c.Query("youtubeId")
	}

	if youtubeId == "" || id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id and youtubeId are required",
		})
	}

	err := youtube.SetYoutubeId(id, youtubeId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}
