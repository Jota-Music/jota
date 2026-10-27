package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// myPlaylistsKey is the cache entry of the signed-in account's playlists.
const (
	myPlaylistsKey         = "me:playlists"
	myPlaylistsSnapshotKey = "me:playlists:page"
)

// myPlaylistsFn is the account browse, swapped in tests so caching can be
// checked without reaching YouTube.
var myPlaylistsFn = fetchMyPlaylists

// errNoPlaylists reports a browse response that carried no playlist at all. It
// is returned from the fetch so the empty list is never cached: a feed whose
// shape we failed to parse would otherwise leave the library empty for good.
var errNoPlaylists = errors.New("no playlists in my playlists response")

// errSession reports that YouTube did not accept the stored cookies as a
// signed-in session, which is what the partial set the sign-in window can read
// looks like to it.
var errSession = errors.New("the YouTube session cannot open private resources: import a cookies.txt export")

// GetMyYouTubePlaylists returns the playlists of the signed-in account. It
// requires account cookies (HasAuth). Results are cached; call
// RevalidateMyYouTubePlaylists to refresh.
func (s *Service) GetMyYouTubePlaylists() ([]music.PlaylistSummary, error) {
	snapshot := playlistSnapshot()
	if !HasAuth() {
		if len(snapshot) > 0 {
			return snapshot, nil
		}
		return nil, errors.New("sign in to YouTube required")
	}

	// A session handed over by the sign-in window carries the signing cookies
	// but not the HttpOnly ones YouTube needs, so asking innertube would only
	// earn a 401. The list that window imported from its own page stands in.
	if sessionValue(cookies()) == "" {
		if len(snapshot) > 0 {
			return snapshot, nil
		}
		return nil, errSession
	}

	list, err := kv.Cached(playlistBucket, myPlaylistsKey, myPlaylistsFn)
	if err == nil {
		return list, nil
	}
	// Whatever the fetch failed at, the page the window imported is still the
	// account's last known list.
	if len(snapshot) > 0 {
		return snapshot, nil
	}
	if errors.Is(err, errNoPlaylists) {
		return []music.PlaylistSummary{}, nil
	}
	return nil, err
}

// myPlaylistsPageURL is the account's playlists page, the one the web app loads
// and the sign-in window already imports. It is a variable so tests can point
// it at a local server.
var myPlaylistsPageURL = "https://www.youtube.com/feed/playlists"

func fetchMyPlaylists() ([]music.PlaylistSummary, error) {
	// The page is the same document the sign-in window imports, so the parse that
	// is already proven there is reused instead of trusting the browse renderers.
	if list, err := pageMyPlaylists(); err == nil {
		log.Printf("youtube: my playlists answered by the page: %d", len(list))
		return list, nil
	} else {
		log.Printf("youtube: my playlists page failed, falling back to browse: %v", err)
	}

	list, firstErr := browseMyPlaylists("FEplaylists", "")
	if firstErr == nil {
		log.Printf("youtube: my playlists answered by browse: %d", len(list))
		return list, nil
	}

	// YouTube moves the library link between releases, and the hardcoded one then
	// answers "Requested entity was not found" even for a valid session. The guide
	// is where the web app itself reads that link from, so follow it.
	target, params, targetErr := playlistsTarget()
	log.Printf("youtube: my playlists browse failed (code %d: %v), guide target %q params %q: %v",
		codeOf(firstErr), firstErr, target, params, targetErr)
	if targetErr != nil {
		return nil, firstErr
	}
	if target == "FEplaylists" && params == "" {
		return nil, firstErr
	}

	list, secondErr := browseMyPlaylists(target, params)
	if secondErr == nil {
		log.Printf("youtube: my playlists answered by browse with the guide's link: %d", len(list))
		return list, nil
	}
	log.Printf("youtube: my playlists browse with the guide's link failed (code %d): %v", codeOf(secondErr), secondErr)
	return nil, secondErr
}

// codeOf returns the innertube error code an error carries, 0 when it is not an
// API error.
func codeOf(err error) int {
	var api *apiError
	if errors.As(err, &api) {
		return api.Code
	}
	return 0
}

// pageMyPlaylists reads the account's playlists from the page the web app
// serves, signed with the stored session.
func pageMyPlaylists() ([]music.PlaylistSummary, error) {
	page, err := fetchPlaylistsPage()
	if err != nil {
		return nil, err
	}
	list := extractMyPlaylistsFromPage(page)
	if len(list) == 0 {
		return nil, errNoPlaylists
	}
	return list, nil
}

// fetchPlaylistsPage downloads the playlists page. A signed GET is what the
// browser does: SAPISIDHASH plus the account cookies.
func fetchPlaylistsPage() ([]byte, error) {
	req, err := http.NewRequest("GET", myPlaylistsPageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	for k, v := range authHeaders("https://www.youtube.com") {
		req.Header.Set(k, v)
	}

	resp, err := visitorClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("my playlists page status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16<<20))
}

// browseMyPlaylists reads one library browse and answers the playlists it found.
func browseMyPlaylists(browseId, params string) ([]music.PlaylistSummary, error) {
	payload := map[string]any{
		"context":  map[string]any{"client": clientContext(webClient())},
		"browseId": browseId,
	}
	if params != "" {
		payload["params"] = params
	}

	data, err := postBrowseWithClient(webClient(), payload)
	if err != nil {
		var api *apiError
		if errors.As(err, &api) && api.Code == http.StatusUnauthorized {
			return nil, errSession
		}
		return nil, fmt.Errorf("my playlists browse failed: %w", err)
	}

	list := extractMyPlaylists(data)
	if len(list) == 0 {
		return nil, errNoPlaylists
	}
	return list, nil
}

// guideEndpoint is the navigation tree the web app builds its own library links
// from.
const guideEndpoint = "https://www.youtube.com/youtubei/v1/guide"

// playlistsTarget asks the guide where the account's playlists live.
func playlistsTarget() (browseId, params string, err error) {
	payload := map[string]any{"context": map[string]any{"client": clientContext(webClient())}}
	data, err := browseResult(retryRequest(webClient(), guideEndpoint, payload, 2))
	if err != nil {
		return "", "", err
	}
	browseId, params, ok := findPlaylistsTarget(data)
	if !ok {
		return "", "", errors.New("the guide carries no playlists entry")
	}
	return browseId, params, nil
}

// findPlaylistsTarget walks a guide response for the endpoint that opens the
// account's playlists.
func findPlaylistsTarget(data []byte) (browseId, params string, ok bool) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return "", "", false
	}

	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			if ok {
				return
			}
			if endpoint, has := val["browseEndpoint"].(map[string]any); has {
				if id, _ := endpoint["browseId"].(string); strings.HasPrefix(id, "FEplaylists") {
					params, _ = endpoint["params"].(string)
					browseId, ok = id, true
					return
				}
			}
			for _, vv := range val {
				walk(vv)
			}
		case []any:
			for _, vv := range val {
				walk(vv)
			}
		}
	}
	walk(root)
	return browseId, params, ok
}

// ImportMyPlaylists stores the playlists of the page the sign-in window fetched
// while signed in, and drops any list cached for another account. It reports how
// many playlists the page carried.
func ImportMyPlaylists(page []byte) (int, error) {
	list := extractMyPlaylistsFromPage(page)
	if len(list) == 0 {
		return 0, nil
	}
	if err := playlistBucket.SetObject(myPlaylistsSnapshotKey, list); err != nil {
		return 0, err
	}
	// The cached list belongs to the account that was signed in before.
	_ = playlistBucket.Delete(myPlaylistsKey)
	return len(list), nil
}

// playlistSnapshot returns the playlists imported at sign-in, if any.
func playlistSnapshot() []music.PlaylistSummary {
	var list []music.PlaylistSummary
	if err := playlistBucket.GetObject(myPlaylistsSnapshotKey, &list); err != nil {
		return nil
	}
	return list
}

// RevalidateMyYouTubePlaylists drops the cached list of the signed-in account's
// playlists.
func (s *Service) RevalidateMyYouTubePlaylists() error {
	return playlistBucket.Delete(myPlaylistsKey)
}

// extractMyPlaylists walks an innertube browse response and collects every
// playlist it can find. The feed has changed shape over time — playlist renderers
// carry a playlistId, today's lockupViewModel carries a contentId tagged with
// contentType — so the walk is intentionally tolerant: any map that names a
// playlist becomes an entry.
func extractMyPlaylists(data []byte) []music.PlaylistSummary {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	return walkMyPlaylists(root)
}

// extractMyPlaylistsFromPage reads the playlists out of the page the sign-in
// window fetched. The page embeds ytInitialData, the same tree the browse
// endpoint returns, so the walk below handles it unchanged.
func extractMyPlaylistsFromPage(page []byte) []music.PlaylistSummary {
	marker := bytes.Index(page, []byte("ytInitialData"))
	if marker < 0 {
		return nil
	}
	rest := page[marker:]
	start := bytes.IndexByte(rest, '{')
	if start < 0 {
		return nil
	}

	var root any
	if err := json.NewDecoder(bytes.NewReader(rest[start:])).Decode(&root); err != nil {
		return nil
	}
	return walkMyPlaylists(root)
}

// walkMyPlaylists collects every playlist it can find anywhere in a page tree:
// the feed has changed shape over time, so the walk does not assume one.
func walkMyPlaylists(root any) []music.PlaylistSummary {
	seen := map[string]bool{}
	var out []music.PlaylistSummary

	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			if id := playlistIdOf(val); id != "" && !seen[id] {
				title := firstText(val, "title", "name")
				if title == "" {
					title = lockupText(val, "title")
				}
				if title == "" {
					title = "YouTube playlist"
				}
				cover := firstThumbnail(val)
				owner := firstText(val, "shortBylineText", "byline")
				if owner == "" {
					owner = lockupByline(val)
				}
				out = append(out, music.PlaylistSummary{
					Id:       music.YouTubePrefix + id,
					Name:     title,
					Cover:    cover,
					Subtitle: owner,
				})
				seen[id] = true
			}
			for _, vv := range val {
				walk(vv)
			}
		case []any:
			for _, vv := range val {
				walk(vv)
			}
		}
	}
	walk(root)
	return out
}

// playlistIdOf returns the playlist id a renderer describes: playlistId on the
// older playlist renderers, contentId on a lockupViewModel. The id pattern is
// what tells them apart from a video lockup, whose id is 11 characters long.
func playlistIdOf(m map[string]any) string {
	for _, key := range []string{"playlistId", "contentId"} {
		if id, ok := m[key].(string); ok && playlistIdPattern.MatchString(id) {
			return id
		}
	}
	return ""
}

// lockupText reads a text field of a lockupViewModel, which nests its texts
// under metadata.lockupMetadataViewModel instead of keeping them at the top.
func lockupText(m map[string]any, key string) string {
	meta, ok := m["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	lockup, ok := meta["lockupMetadataViewModel"].(map[string]any)
	if !ok {
		return ""
	}
	return firstText(lockup, key)
}

// lockupByline reads the owner of a lockupViewModel: its metadata rows live
// under metadata.lockupMetadataViewModel.metadata.contentMetadataViewModel,
// and the first part of the first row is the byline.
func lockupByline(m map[string]any) string {
	rows := lockupRows(m)
	if len(rows) == 0 {
		return ""
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		return ""
	}
	parts, ok := row["metadataParts"].([]any)
	if !ok || len(parts) == 0 {
		return ""
	}
	part, ok := parts[0].(map[string]any)
	if !ok {
		return ""
	}
	return firstText(part, "text")
}

// lockupRows returns the metadata rows of a lockupViewModel, which nests them
// under metadata.lockupMetadataViewModel.metadata.contentMetadataViewModel.
func lockupRows(m map[string]any) []any {
	meta, ok := m["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	lockup, ok := meta["lockupMetadataViewModel"].(map[string]any)
	if !ok {
		return nil
	}
	metadata, ok := lockup["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	content, ok := metadata["contentMetadataViewModel"].(map[string]any)
	if !ok {
		return nil
	}
	rows, _ := content["metadataRows"].([]any)
	return rows
}

// firstText returns the first non-empty text found under any of the given keys,
// handling both "simpleText" and "runs" shapes.
func firstText(m map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := m[key]
		if !ok {
			continue
		}
		if s, ok := v.(string); ok && s != "" {
			return s
		}
		mv, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := mv["simpleText"].(string); ok && s != "" {
			return s
		}
		if s, ok := mv["content"].(string); ok && s != "" {
			return s
		}
		if runs, ok := mv["runs"].([]any); ok {
			for _, r := range runs {
				if rm, ok := r.(map[string]any); ok {
					if s, ok := rm["text"].(string); ok && s != "" {
						return s
					}
				}
			}
		}
		if inner, ok := mv["text"].(map[string]any); ok {
			if s, ok := inner["simpleText"].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// firstThumbnail returns the last (largest) thumbnail URL found anywhere under m,
// taking both the thumbnails list and the sources list of a thumbnailViewModel.
func firstThumbnail(m map[string]any) string {
	var found string
	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			for _, key := range []string{"thumbnails", "sources"} {
				list, ok := val[key].([]any)
				if !ok || len(list) == 0 {
					continue
				}
				if last, ok := list[len(list)-1].(map[string]any); ok {
					if u, ok := last["url"].(string); ok && u != "" {
						found = absolute(u)
					}
				}
			}
			for _, vv := range val {
				walk(vv)
			}
		case []any:
			for _, vv := range val {
				walk(vv)
			}
		}
	}
	walk(m)
	if found != "" {
		found = capThumbnail(found)
	}
	return found
}
