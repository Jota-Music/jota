package youtube

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const repoBase = "https://github.com/yt-dlp/yt-dlp/releases/latest/download"

func getBinaryName() string {
	if runtime.GOOS == "windows" {
		return "yt-dlp.exe"
	}
	return "yt-dlp"
}

func findGlobal() (string, bool) {
	path, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", false
	}
	return path, true
}

func getInstallPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(cacheDir, "yt-dlp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, getBinaryName()), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed download: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func Ensure() (string, error) {
	if path, ok := findGlobal(); ok {
		return path, nil
	}

	path, err := getInstallPath()
	if err != nil {
		return "", err
	}

	if fileExists(path) {
		return path, nil
	}

	fmt.Println("yt-dlp not found, downloading...")

	url := repoBase + "/" + getBinaryName()

	if err := downloadFile(url, path); err != nil {
		return "", err
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0755); err != nil {
			return "", err
		}
	}

	return path, nil
}
