package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"jota/server/internal/kv"

	"github.com/gofiber/fiber/v3"
)

const (
	// CookieName is the HTTP-only session cookie sent to the client.
	CookieName = "jota_sid"
	// LocalsUserName is the Fiber Locals key for the authenticated user's name.
	LocalsUserName = "userName"
)

var (
	sessionBucket = kv.UseBucket("session")

	// IdleTTL is sliding inactivity window: each validated request extends it.
	IdleTTL = 7 * 24 * time.Hour
	// AbsoluteMax is the maximum session lifetime since login.
	AbsoluteMax = 30 * 24 * time.Hour

	errSessionNotFound = errors.New("session not found")
)

// Record is stored server-side; the cookie only holds an opaque session id.
type Record struct {
	UserName  string    `json:"userName"`
	CreatedAt time.Time `json:"createdAt"`
}

func newSessionID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// Create persists a new session and returns its opaque id.
func Create(userName string) (string, error) {
	id, err := newSessionID()
	if err != nil {
		return "", err
	}
	rec := Record{
		UserName:  userName,
		CreatedAt: time.Now().UTC(),
	}
	if err := sessionBucket.SetObject(id, rec, IdleTTL); err != nil {
		return "", err
	}
	return id, nil
}

// Delete removes a session from the store (logout).
func Delete(sessionID string) error {
	return sessionBucket.Delete(sessionID)
}

// ValidateAndTouch loads the session, enforces absolute max lifetime, and
// refreshes the Badger TTL (sliding idle expiration).
func ValidateAndTouch(sessionID string) (userName string, err error) {
	if sessionID == "" {
		return "", errSessionNotFound
	}
	var rec Record
	if err := sessionBucket.GetObject(sessionID, &rec); err != nil {
		return "", errSessionNotFound
	}
	if time.Since(rec.CreatedAt) > AbsoluteMax {
		_ = sessionBucket.Delete(sessionID)
		return "", errSessionNotFound
	}
	if err := sessionBucket.SetObject(sessionID, rec, IdleTTL); err != nil {
		return "", err
	}
	return rec.UserName, nil
}

// WriteCookie sets the HTTP-only session cookie on the response.
func WriteCookie(c fiber.Ctx, sessionID string) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(IdleTTL.Seconds()),
		HTTPOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: "Lax",
	})
}

// ClearCookie removes the session cookie from the client.
func ClearCookie(c fiber.Ctx) {
	c.ClearCookie(CookieName)
}

// Require returns middleware that validates the session, slides expiration,
// and sets LocalsUserName for downstream handlers.
func Require() fiber.Handler {
	return func(c fiber.Ctx) error {
		sid := c.Cookies(CookieName)
		name, err := ValidateAndTouch(sid)
		if err != nil {
			ClearCookie(c)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		c.Locals(LocalsUserName, name)
		return c.Next()
	}
}
