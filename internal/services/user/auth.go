package user

import (
	"errors"
	"jota/server/internal/auth"
	"jota/server/internal/kv"
	"log"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

var userBucket = kv.UseBucket("user")

func (s *UserService) NewUser(newUser auth.User) (auth.User, error) {
	var oldUser auth.User
	userError := userBucket.GetObject(newUser.Name, &oldUser)
	if userError == nil {
		return auth.User{}, errors.New("user already exists")
	}

	userBucket.SetObject(newUser.Name, newUser)

	return newUser, nil
}

func (s *UserService) GetUser(name string) (auth.User, error) {
	var user auth.User
	userError := userBucket.GetObject(name, &user)
	if userError != nil {
		return auth.User{}, userError
	}

	log.Println("user", user)

	return user, nil
}
