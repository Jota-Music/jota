package env

import "os"

type Config struct {
	SpotifyClientID string
	// DiscordClientID pins the Discord application used for rich presence. The
	// default is Jota's registered app; forks override it with DISCORD_CLIENT_ID.
	DiscordClientID string
	// RelayAPIURL and RelayAPIToken pin the relay a developer tests against,
	// overriding whatever relay the user saved. Meant for local/dev runs.
	RelayAPIURL   string
	RelayAPIToken string
}

func Load() Config {
	discordClientID := os.Getenv("DISCORD_CLIENT_ID")
	if discordClientID == "" {
		discordClientID = "1550829418388131892"
	}

	return Config{
		SpotifyClientID: os.Getenv("SPOTIFY_CLIENT_ID"),
		DiscordClientID: discordClientID,
		RelayAPIURL:     os.Getenv("RELAY_API_URL"),
		RelayAPIToken:   os.Getenv("RELAY_API_TOKEN"),
	}
}
