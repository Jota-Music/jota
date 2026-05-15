package youtube

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

var youtubeAuthCookies = []string{
	"__Secure-3PSID",
	"SAPISID",
	"APISID",
	"HSID",
	"SSID",
	"__Secure-3PAPISID",
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

	data, err := os.ReadFile(cookiesPath())
	if err != nil {
		return fmt.Sprintf("cookies file at storage/cookies.txt (%d bytes, unreadable: %v)", info.Size(), err)
	}

	var found []string
	for _, name := range youtubeAuthCookies {
		if strings.Contains(string(data), name) {
			found = append(found, name)
		}
	}

	if len(found) == 0 {
		return fmt.Sprintf("cookies file at storage/cookies.txt (%d bytes) - WARNING: no YouTube auth cookies found (expected one of: %s)", info.Size(), strings.Join(youtubeAuthCookies, ", "))
	}

	return fmt.Sprintf("using cookies from storage/cookies.txt (%d bytes, auth cookies: %s)", info.Size(), strings.Join(found, ", "))
}
