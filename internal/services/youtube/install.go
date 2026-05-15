package youtube

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
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

func binaryAge(path string) time.Duration {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return time.Since(info.ModTime())
}

func Ensure() (string, error) {
	path, err := getInstallPath()
	if err != nil {
		if globalPath, ok := findGlobal(); ok {
			return globalPath, nil
		}
		return "", err
	}

	needsDownload := true
	if fileExists(path) {
		age := binaryAge(path)
		if age < 24*time.Hour {
			needsDownload = false
		} else {
			fmt.Println("yt-dlp is older than 24h, checking for update...")
		}
	}

	if needsDownload {
		fmt.Println("Downloading latest yt-dlp...")

		url := repoBase + "/" + getBinaryName()
		if err := downloadFile(url, path); err != nil {
			if globalPath, ok := findGlobal(); ok {
				fmt.Println("Download failed, falling back to system yt-dlp")
				return globalPath, nil
			}
			return "", fmt.Errorf("download failed: %w", err)
		}

		if runtime.GOOS != "windows" {
			if err := os.Chmod(path, 0755); err != nil {
				return "", err
			}
		}

		fmt.Println("yt-dlp updated successfully")
	}

	return path, nil
}
