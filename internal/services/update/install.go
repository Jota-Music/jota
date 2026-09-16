package update

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrUnsupported = errors.New("update: automatic install is not supported on this channel")

// Progress reports download progress. Total is 0 when the server sends no
// Content-Length.
type Progress struct {
	Written int64 `json:"written"`
	Total   int64 `json:"total"`
}

// httpClient is used for artifact and checksum downloads. It has no short
// timeout (release assets can be large) but a stalled connection is bounded:
// 30s to start responding, 10m for the whole transfer.
var httpClient = func() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Transport: t, Timeout: 10 * time.Minute}
}()

// Install downloads the newest release for this channel, verifies it against
// the release's SHA256SUMS and swaps it in. onProgress may be nil. On success
// the caller is expected to quit so the scheduled relaunch can start the new
// version.
func Install(current string, onProgress func(Progress)) error {
	ch := channel()
	if ch == External {
		return ErrUnsupported
	}

	releases, err := fetch()
	if err != nil {
		return err
	}
	rel := pick(releases)
	if rel == nil || !newer(strings.TrimPrefix(rel.TagName, "v"), current) {
		return errors.New("update: already up to date")
	}
	a := selectAsset(ch, rel.Assets)
	if a == nil {
		return fmt.Errorf("update: release %s has no asset for %s", rel.TagName, ch)
	}

	dir, err := targetDir()
	if err != nil {
		return err
	}
	staged, err := download(a.URL, dir, func(written, total int64) {
		if onProgress != nil {
			onProgress(Progress{Written: written, Total: total})
		}
	})
	if err != nil {
		return err
	}
	if err := verify(staged, a.Name, rel.Assets); err != nil {
		_ = os.Remove(staged)
		return err
	}
	if err := install(staged); err != nil {
		_ = os.Remove(staged)
		return err
	}
	_ = os.Remove(staged) // darwin leaves the staged archive behind; a no-op elsewhere
	return nil
}

func download(url, dir string, onProgress func(written, total int64)) (string, error) {
	f, err := os.CreateTemp(dir, ".jota-update-*")
	if err != nil {
		return "", fmt.Errorf("update: stage download: %w", err)
	}
	path := f.Name()
	fail := func(err error) (string, error) {
		f.Close()
		os.Remove(path)
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fail(err)
	}
	req.Header.Set("User-Agent", "jota")
	resp, err := httpClient.Do(req)
	if err != nil {
		return fail(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fail(fmt.Errorf("update: download: HTTP %d", resp.StatusCode))
	}

	total := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return fail(werr)
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fail(rerr)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func verify(path, name string, assets []asset) error {
	sums := findAsset(assets, "SHA256SUMS")
	if sums == nil {
		return errors.New("update: release has no SHA256SUMS")
	}
	want, err := digestFromSums(sums.URL, name)
	if err != nil {
		return err
	}
	got, err := fileDigest(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("update: checksum mismatch for %s", name)
	}
	return nil
}

func digestFromSums(url, name string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "jota")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update: SHA256SUMS: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return parseSums(string(body), name)
}

// parseSums reads a `<digest>  <filename>` listing, tolerating the `*` binary
// mode marker and a leading path.
func parseSums(body, name string) (string, error) {
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		file := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(file) == name {
			return fields[0], nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("update: SHA256SUMS has no entry for %s", name)
}

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
