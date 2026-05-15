package user_controller

import (
	"io"
	"os"

	"jota/server/internal/services/youtube"

	"github.com/gofiber/fiber/v3"
)

func UploadCookies(c fiber.Ctx) error {
	file, err := c.FormFile("cookies")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No cookies file provided",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open uploaded file",
		})
	}
	defer src.Close()

	if err := os.MkdirAll("storage", 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create storage directory",
		})
	}

	dst, err := os.Create(youtube.CookiesPath())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save cookies file",
		})
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to write cookies file",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cookies uploaded successfully",
	})
}

func DeleteCookies(c fiber.Ctx) error {
	if err := os.Remove(youtube.CookiesPath()); err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "No cookies file configured",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove cookies file",
		})
	}
	return c.JSON(fiber.Map{
		"message": "Cookies removed successfully",
	})
}

func GetCookiesStatus(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"configured": youtube.HasCookies(),
	})
}
