package auth_controller

import (
	"jota/server/internal/auth"
	"jota/server/internal/repositories"
	"jota/server/internal/session"

	"github.com/gofiber/fiber/v3"
)

func Register(c fiber.Ctx) error {
	body := new(auth.User)
	if err := c.Bind().Body(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	created, err := repositories.Use.Auth.NewUser(body.Name, body.Pass)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User data received",
		"user":    created.Name,
	})
}

func GetUser(c fiber.Ctx) error {
	user, err := repositories.Use.Auth.GetUser(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

func LogIn(c fiber.Ctx) error {
	body := new(auth.User)
	if err := c.Bind().Body(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	user, err := repositories.Use.Auth.GetUser(body.Name)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if err := repositories.Use.Auth.VerifyPasswords(user.Name, body.Pass); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	sid, err := session.Create(user.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create session",
		})
	}
	session.WriteCookie(c, sid)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"user":    user.Name,
	})
}

func LogOut(c fiber.Ctx) error {
	if sid := c.Cookies(session.CookieName); sid != "" {
		_ = session.Delete(sid)
	}
	session.ClearCookie(c)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logged out",
	})
}

func Me(c fiber.Ctx) error {
	name, _ := c.Locals(session.LocalsUserName).(string)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": name,
	})
}
