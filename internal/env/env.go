package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SpotifyClientID string
	// RelayAPIURL and RelayAPIToken pin the relay a developer tests against,
	// overriding whatever relay the user saved. Meant for local/dev runs.
	RelayAPIURL   string
	RelayAPIToken string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using defaults")
	}

	return Config{
		SpotifyClientID: os.Getenv("SPOTIFY_CLIENT_ID"),
		RelayAPIURL:     os.Getenv("RELAY_API_URL"),
		RelayAPIToken:   os.Getenv("RELAY_API_TOKEN"),
	}
}
