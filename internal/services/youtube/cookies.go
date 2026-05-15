package youtube

import (
	"fmt"
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

func CookieStatus() string {
	info, err := os.Stat(cookiesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "no cookies file at storage/cookies.txt"
		}
		return fmt.Sprintf("cookies file error: %v", err)
	}

	if info.Size() == 0 {
		return "cookies file at storage/cookies.txt is empty"
	}

	return fmt.Sprintf("using cookies from storage/cookies.txt (%d bytes)", info.Size())
}
