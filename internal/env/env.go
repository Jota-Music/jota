package env

import (
	"os"
	"strings"
)

type Config struct {
	AllowedOrigins []string
	Port           string
}

func Load() Config {
	return Config{
		AllowedOrigins: getAllowedOrigins(),
		Port:           getPort(),
	}
}

func getAllowedOrigins() []string {
	origins := os.Getenv("ALLOWED_ORIGINS")

	// Fallback dev seguro
	if origins == "" {
		return []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:4173",
			"http://127.0.0.1:4173",
		}
	}

	parts := strings.Split(origins, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "3001"
	}
	return port
}
