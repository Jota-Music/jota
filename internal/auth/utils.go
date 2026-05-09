package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func validatePassword(password string) error {
	if len(password) < 10 {
		return ErrWeakPassword
	}

	var hasLower, hasUpper, hasNumber bool

	for _, c := range password {
		switch {
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsNumber(c):
			hasNumber = true
		}
	}

	if !hasLower || !hasUpper || !hasNumber {
		return ErrWeakPassword
	}

	weakPasswords := map[string]bool{
		"1234567890":  true,
		"password":    true,
		"Password123": true,
		"qwerty1234":  true,
		"1111111111":  true,
	}

	if weakPasswords[password] {
		return ErrWeakPassword
	}

	return nil
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func ComparePasswords(hashedPassword string, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return errors.New("invalid password")
	}
	return nil
}
