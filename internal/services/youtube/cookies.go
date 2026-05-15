package youtube

import (
	"os"
	"path/filepath"
)

func cookiesPath() string {
	return filepath.Join("storage", "cookies.txt")
}

func HasCookies() bool {
	_, err := os.Stat(cookiesPath())
	return err == nil
}

func CookiesPath() string {
	return cookiesPath()
}
