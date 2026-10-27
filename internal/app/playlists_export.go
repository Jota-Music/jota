package app

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// exportWorkers caps how many playlists are resolved at once. Each resolution is
// a network round trip, so a handful of them keeps the export fast without
// hammering the sources.
const exportWorkers = 3

// The actors below the catalog take a context.Background(), so a source that
// never answers would pin a worker and the whole save with it. Bound both: one
// playlist, and the pass as a whole. Whatever is left unresolved keeps the copy
// the document already carried.
// shortcut: the calls are not cancellable, so a timed-out request keeps running
// in the background until the source gives up; revisit when the catalog takes a
// context.
const (
	playlistTimeout = 20 * time.Second
	snapshotBudget  = 2 * time.Minute
)

// resolve asks the catalog for one playlist, giving up after playlistTimeout.
func (a *App) resolve(id string) (music.Playlist, error) {
	type answer struct {
		playlist music.Playlist
		err      error
	}
	done := make(chan answer, 1)
	go func() {
		pl, err := a.Catalog.GetFullPlaylist(id)
		done <- answer{pl, err}
	}()
	select {
	case got := <-done:
		return got.playlist, got.err
	case <-time.After(playlistTimeout):
		return music.Playlist{}, fmt.Errorf("no answer in %v", playlistTimeout)
	}
}

// ExternalProgress is the export progress event: how many playlists have been
// resolved, which one is in flight, and how many were skipped.
type ExternalProgress struct {
	Done   int    `json:"done"`
	Total  int    `json:"total"`
	Name   string `json:"name,omitempty"`
	Failed int    `json:"failed"`
}

// externalSummaries lists the playlists of every connected source plus the ones
// the user saved, deduplicated by id so a saved playlist is exported once.
func (a *App) externalSummaries() []music.PlaylistSummary {
	var lists [][]music.PlaylistSummary

	if a.Spotify.IsConnected() || a.Spotify.IsDegraded() {
		if user := a.Spotify.Username(); user != "" {
			list, err := a.Spotify.GetUserPlaylists(user)
			if err != nil {
				log.Printf("export: spotify playlists: %v", err)
			}
			lists = append(lists, list)
		}
	}

	if youtube.HasAuth() {
		list, err := a.YouTube.GetMyYouTubePlaylists()
		if err != nil {
			log.Printf("export: youtube playlists: %v", err)
		}
		lists = append(lists, list)
	}

	if list, err := a.Saved.List(); err != nil {
		log.Printf("export: saved playlists: %v", err)
	} else {
		lists = append(lists, list)
	}

	return dedupe(lists)
}

// dedupe joins the per-source listings keeping the first occurrence of each id.
func dedupe(lists [][]music.PlaylistSummary) []music.PlaylistSummary {
	seen := make(map[string]struct{})
	out := make([]music.PlaylistSummary, 0)
	for _, list := range lists {
		for _, summary := range list {
			if _, dup := seen[summary.Id]; dup {
				continue
			}
			seen[summary.Id] = struct{}{}
			out = append(out, summary)
		}
	}
	return out
}

// externalPlaylists resolves every summary into a playlist snapshot. A playlist
// that fails or does not answer in time is logged and skipped: it must not abort
// the export nor shrink the document, because the caller keeps the copy it
// already had for whatever comes back missing. progress reports each one to the
// UI, which only the export does: a background sync must stay silent.
func (a *App) externalPlaylists(summaries []music.PlaylistSummary, progress bool) []playlists.Snapshot {
	if len(summaries) == 0 {
		return nil
	}

	deadline := time.Now().Add(snapshotBudget)
	queue := make(chan int, len(summaries))
	for i := range summaries {
		queue <- i
	}
	close(queue)

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]*playlists.Snapshot, len(summaries))
		done    int
		failed  int
	)

	for range min(exportWorkers, len(summaries)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				summary := summaries[i]

				var pl music.Playlist
				var err error
				if time.Now().Before(deadline) {
					pl, err = a.resolve(summary.Id)
				} else {
					err = fmt.Errorf("out of time for the whole pass")
				}

				name := summary.Name
				if err == nil && pl.Name != "" {
					name = pl.Name
				}

				mu.Lock()
				done++
				if err != nil {
					failed++
					log.Printf("export: keeping the stored copy of %q: %v", summary.Id, err)
				} else {
					results[i] = &playlists.Snapshot{
						Id:     summary.Id,
						Source: source(summary.Id),
						Name:   name,
						Owner:  pl.Owner,
						Songs:  pl.Songs,
					}
				}
				update := ExternalProgress{Done: done, Total: len(summaries), Name: name, Failed: failed}
				mu.Unlock()

				if progress {
					application.Get().Event.Emit("playlists:export", update)
				}
			}
		}()
	}
	wg.Wait()

	out := make([]playlists.Snapshot, 0, len(summaries))
	for _, pl := range results {
		if pl != nil {
			out = append(out, *pl)
		}
	}
	return out
}

// source names the owner of a playlist id.
func source(id string) string {
	switch {
	case strings.HasPrefix(id, music.YouTubePrefix):
		return "youtube"
	case strings.HasPrefix(id, music.LocalPrefix):
		return "local"
	default:
		return "spotify"
	}
}
