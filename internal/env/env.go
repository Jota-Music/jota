package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SpotifyClientID string
	TURNURL         string
	TURNUser        string
	TURNPass        string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using defaults")
	}

	return Config{
		SpotifyClientID: os.Getenv("SPOTIFY_CLIENT_ID"),
		TURNURL:         os.Getenv("TURN_URL"),
		TURNUser:        os.Getenv("TURN_USER"),
		TURNPass:        os.Getenv("TURN_PASS"),
	}
}
