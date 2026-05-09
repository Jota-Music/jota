package auth

import (
	"errors"
)

var (
	ErrWeakPassword = errors.New("password does not meet security requirements")
)

type Uses interface {
	NewUser(user User) (User, error)
	GetUser(id string) (User, error)
}

type AuthRepository struct {
	use Uses
}

func NewAuthRepository(use Uses) *AuthRepository {
	return &AuthRepository{
		use: use,
	}
}

func (m *AuthRepository) GetUser(id string) (User, error) {
	return m.use.GetUser(id)
}

func (m *AuthRepository) NewUser(name string, password string) (User, error) {
	// if err := validatePassword(password); err != nil {
	// 	return User{}, err
	// }

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	user := User{
		Name: name,
		Pass: string(hashedPassword),
	}

	return m.use.NewUser(user)
}

func (m *AuthRepository) VerifyPasswords(userId string, tryPassword string) error {
	user, err := m.use.GetUser(userId)
	if err != nil {
		return err
	}

	return ComparePasswords(user.Pass, tryPassword)
}
