package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SpotifyClientID string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using defaults")
	}

	return Config{
		SpotifyClientID: os.Getenv("SPOTIFY_CLIENT_ID"),
	}
}
