//go:build linux

package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// archToken mirrors the AppImage naming produced by build/linux/appimage/build.sh.
func archToken() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}

// releaseServer stands up a fake GitHub releases API plus the asset and its
// SHA256SUMS, so Install can run against something that behaves like the real
// release.
func releaseServer(t *testing.T, version, assetName string, payload []byte, sum string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/repos/"+repo+"/releases", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `[{"tag_name":"v%s","html_url":"%s/releases/v%s","assets":[`+
			`{"name":%q,"browser_download_url":"%s/asset"},`+
			`{"name":"SHA256SUMS","browser_download_url":"%s/sums"}]}]`,
			version, srv.URL, version, assetName, srv.URL, srv.URL)
	})
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", sum, assetName)
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func stub(t *testing.T, srv *httptest.Server) {
	t.Helper()
	oldBase, oldRelaunch := apiBase, relaunch
	apiBase = srv.URL
	relaunch = func(string) error { return nil }
	t.Cleanup(func() {
		apiBase = oldBase
		relaunch = oldRelaunch
	})
	t.Setenv("NO_PROXY", "127.0.0.1,localhost")
}

func assertNoStaging(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".jota-update-") {
			t.Errorf("leftover staging file: %s", e.Name())
		}
	}
}

func TestInstallAppImage(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "jota-"+archToken()+".AppImage")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", target)

	payload := []byte("NEW APPIMAGE BYTES")
	h := sha256.Sum256(payload)
	srv := releaseServer(t, "0.2.0", filepath.Base(target), payload, hex.EncodeToString(h[:]))
	stub(t, srv)

	info, err := Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available || !info.Installable || info.Latest != "0.2.0" {
		t.Fatalf("Check = %+v; want available+installable 0.2.0", info)
	}

	var last Progress
	if err := Install("0.1.0", func(p Progress) { last = p }); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if last.Written != int64(len(payload)) {
		t.Fatalf("progress written = %d; want %d", last.Written, len(payload))
	}
	if last.Total > 0 && last.Total != int64(len(payload)) {
		t.Fatalf("progress total = %d; want %d", last.Total, len(payload))
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("target = %q; want %q", got, payload)
	}
	fi, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o100 == 0 {
		t.Fatalf("target not executable: %v", fi.Mode())
	}
	assertNoStaging(t, dir)
}

func TestInstallRejectsBadChecksum(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "jota-"+archToken()+".AppImage")
	old := []byte("OLD")
	if err := os.WriteFile(target, old, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", target)

	srv := releaseServer(t, "0.2.0", filepath.Base(target), []byte("TAMPERED"),
		strings.Repeat("0", 64))
	stub(t, srv)

	err := Install("0.1.0", nil)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("Install err = %v; want checksum mismatch", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != string(old) {
		t.Fatal("target was modified despite a checksum failure")
	}
	assertNoStaging(t, dir)
}

func TestInstallSkipsWhenCurrent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "jota-"+archToken()+".AppImage")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", target)

	srv := releaseServer(t, "0.2.0", filepath.Base(target), []byte("NEW"), strings.Repeat("0", 64))
	stub(t, srv)

	if err := Install("0.2.0", nil); err == nil {
		t.Fatal("Install on the current version should not run")
	}
}
