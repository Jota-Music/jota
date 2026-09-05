package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	PublicURL     string
	SuperUsername string
	SuperPassword string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using defaults")
	}

	return Config{
		Port:          getPort(),
		PublicURL:     os.Getenv("PUBLIC_URL"),
		SuperUsername: os.Getenv("SUPER_USERNAME"),
		SuperPassword: os.Getenv("SUPER_PASSWORD"),
	}
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "3001"
	}
	return port
}
