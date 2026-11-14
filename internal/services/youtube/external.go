package youtube

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// externalTimeout bounds the whole yt-dlp run: it downloads the watch page and
// the player script, then solves the challenge, which takes seconds on a good
// network and never finishes on a broken one.
const externalTimeout = 45 * time.Second

// errExternal reports that the fallback could not answer, never that the video
// is unavailable: the caller decides what to tell the user.
var errExternal = errors.New("yt-dlp fallback failed")

// externalAudio asks yt-dlp for a playable URL when the app's own resolve ends
// up with a stream it cannot open. It covers the videos YouTube only serves to
// a signed client: members-only ones, whose every URL carries the player's `n`
// challenge, which needs the player script interpreted to answer.
//
// It is opportunistic: without yt-dlp in PATH nothing changes, and the error
// the app already had is what the user sees. yt-dlp needs a JS runtime (deno,
// node or bun) for the challenge, so a machine without both still cannot play
// these videos.
func externalAudio(videoID string) (string, clientConfig, error) {
	path, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", clientConfig{}, errors.New("yt-dlp is not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), externalTimeout)
	defer cancel()

	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-warnings",
		"--no-progress",
		"-f", "bestaudio/best",
		"-g",
		"https://www.youtube.com/watch?v=" + videoID,
	}
	if raw := cookies(); raw != "" {
		dir, err := os.MkdirTemp("", "jota-cookies")
		if err != nil {
			return "", clientConfig{}, err
		}
		defer os.RemoveAll(dir)

		file := filepath.Join(dir, "cookies.txt")
		if err := os.WriteFile(file, []byte(netscape(raw)), 0o600); err != nil {
			return "", clientConfig{}, err
		}
		args = append([]string{"--cookies", file}, args...)
	}

	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(exit.Stderr) > 0 {
			return "", clientConfig{}, fmt.Errorf("%w: %s", errExternal, brief(string(exit.Stderr)))
		}
		return "", clientConfig{}, fmt.Errorf("%w: %v", errExternal, err)
	}

	raw := firstURL(string(out))
	if raw == "" {
		return "", clientConfig{}, fmt.Errorf("%w: no URL in the answer", errExternal)
	}
	return raw, webClient(), nil
}

// firstURL takes the first https line of yt-dlp's answer. -g prints one URL per
// format, and a request that matches several is answered with all of them.
func firstURL(out string) string {
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "https://") {
			return line
		}
	}
	return ""
}

// netscape writes the stored Cookie header as a cookies.txt, the shape yt-dlp
// reads. Every cookie is scoped to youtube.com: the header came from the app's
// own session, which is exactly the one YouTube signs those requests with.
func netscape(raw string) string {
	var b strings.Builder
	b.WriteString("# Netscape HTTP Cookie File\n")
	for part := range strings.SplitSeq(raw, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || name == "" {
			continue
		}
		fmt.Fprintf(&b, ".youtube.com\tTRUE\t/\tTRUE\t%d\t%s\t%s\n", 0, name, value)
	}
	return b.String()
}

// brief keeps the reason of a failed run readable: yt-dlp explains in one line
// and then lists the clients it tried.
func brief(stderr string) string {
	for line := range strings.SplitSeq(stderr, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "ERROR:"))
		if line != "" {
			if len(line) > 200 {
				line = line[:200]
			}
			return line
		}
	}
	return "no reason"
}
