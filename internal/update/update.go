package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

const repo = "Jota-Music/jota"

// apiBase is overridable in tests.
var apiBase = "https://api.github.com"

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Draft   bool    `json:"draft"`
	Assets  []asset `json:"assets"`
}

func channel() Channel {
	if os.Getenv("APPIMAGE") != "" {
		return AppImage
	}
	switch runtime.GOOS {
	case "darwin":
		return MacOS
	case "windows":
		return Windows
	default:
		return External
	}
}

// Check reports the newest release and whether this build can install it.
func Check(current string) (Info, error) {
	ch := channel()
	info := Info{Current: current}
	releases, err := fetch()
	if err != nil {
		return info, err
	}
	rel := pick(releases)
	if rel == nil {
		return info, nil
	}
	info.Latest = strings.TrimPrefix(rel.TagName, "v")
	info.URL = rel.HTMLURL
	info.Available = newer(info.Latest, current)
	info.Installable = info.Available && selectAsset(ch, rel.Assets) != nil
	return info, nil
}

func fetch() ([]release, error) {
	req, err := http.NewRequest(http.MethodGet,
		apiBase+"/repos/"+repo+"/releases?per_page=30", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "jota")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}
	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func pick(releases []release) *release {
	var best *release
	var bestVer [3]int
	for i := range releases {
		r := &releases[i]
		if r.Draft {
			continue
		}
		v, ok := parse(r.TagName)
		if !ok {
			continue
		}
		if best == nil || slices.Compare(bestVer[:], v[:]) < 0 {
			best, bestVer = r, v
		}
	}
	return best
}

// selectAsset picks the artifact this build can install, or nil when the
// channel has no self-update path.
func selectAsset(ch Channel, assets []asset) *asset {
	for i := range assets {
		if matches(ch, assets[i].Name) {
			return &assets[i]
		}
	}
	return nil
}

func matches(ch Channel, name string) bool {
	n := strings.ToLower(name)
	switch ch {
	case AppImage:
		return strings.HasSuffix(n, ".appimage") && containsArch(n, runtime.GOARCH)
	case MacOS:
		return n == "jota-macos.zip"
	case Windows:
		return strings.HasSuffix(n, ".exe") && strings.Contains(n, "windows") &&
			!strings.Contains(n, "installer") && containsArch(n, runtime.GOARCH)
	}
	return false
}

func containsArch(name, arch string) bool {
	if strings.Contains(name, arch) {
		return true
	}
	switch arch {
	case "amd64":
		return strings.Contains(name, "x86_64") || strings.Contains(name, "x64")
	case "arm64":
		return strings.Contains(name, "aarch64")
	}
	return false
}

func findAsset(assets []asset, name string) *asset {
	for i := range assets {
		if strings.EqualFold(assets[i].Name, name) {
			return &assets[i]
		}
	}
	return nil
}

func newer(latest, current string) bool {
	l, ok := parse(latest)
	if !ok {
		return false
	}
	c, ok := parse(current)
	if !ok {
		return false
	}
	return slices.Compare(c[:], l[:]) < 0
}

// parse reads the numeric MAJOR.MINOR.PATCH prefix of a tag, tolerating a
// leading "v" and ignoring any "-rc"/"+build" suffix.
// ponytail: numeric triple only, switch to a real semver lib if prereleases matter.
func parse(tag string) ([3]int, bool) {
	var out [3]int
	tag = strings.TrimPrefix(tag, "v")
	if i := strings.IndexAny(tag, "-+"); i >= 0 {
		tag = tag[:i]
	}
	parts := strings.Split(tag, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
