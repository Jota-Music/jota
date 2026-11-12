package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/state"

	"github.com/dgraph-io/badger/v4"
)

// legacyDir is where the old Badger store lived.
func legacyDir() (string, error) {
	base, err := state.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "jota", "storage", "kv"), nil
}

// HasLegacyStore reports whether an old Badger store is present and worth
// migrating. The caller (frontend or startup) uses it to decide whether to
// offer the import.
func HasLegacyStore() bool {
	dir, err := legacyDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "MANIFEST"))
	return err == nil
}

// MigrateLegacyStore reads the old Badger store and writes its contents into
// the new state.json, then moves the old directory aside so it is not migrated
// twice. It returns a short summary of what was imported.
//
// The cache is NOT migrated: tracks, artists, albums and the rest are
// re-fetched the next time the source answers. Only the device's own state
// (playlists, follows, ordering, rooms, saved, settings, sessions, spotify,
// youtube, window geometry) is carried over.
func MigrateLegacyStore() (string, error) {
	dir, err := legacyDir()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(dir, "MANIFEST")); err != nil {
		return "no legacy store found", nil
	}

	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil))
	if err != nil {
		return "", fmt.Errorf("cannot open legacy store: %w", err)
	}

	var imported int
	buckets := map[string]map[string][]byte{}
	err = db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			bucket, key, ok := strings.Cut(string(item.KeyCopy(nil)), ":")
			if !ok {
				continue
			}
			data, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			if buckets[bucket] == nil {
				buckets[bucket] = map[string][]byte{}
			}
			buckets[bucket][key] = data
			imported++
		}
		return nil
	})
	if err != nil {
		db.Close()
		return "", fmt.Errorf("cannot read legacy store: %w", err)
	}

	// --- Transform and write to the new state ---

	// Playlists: index + p:<id> + graves -> playlists section.
	if b, ok := buckets["playlists"]; ok {
		var ids []string
		if data, ok := b["index"]; ok {
			_ = json.Unmarshal(data, &ids)
		}
		var playlists []map[string]any
		for _, id := range ids {
			if data, ok := b["p:"+id]; ok {
				var pl map[string]any
				if err := json.Unmarshal(data, &pl); err == nil {
					playlists = append(playlists, pl)
				}
			}
		}
		graves := map[string]int64{}
		if data, ok := b["graves"]; ok {
			_ = json.Unmarshal(data, &graves)
		}
		section := map[string]any{
			"playlists": playlists,
			"order":     ids,
			"at":        time.Now().UnixMilli(),
		}
		if len(graves) > 0 {
			section["graves"] = graves
		}
		if err := state.Write(state.Playlists, section); err != nil {
			log.Printf("migrate: playlists: %v", err)
		}
	}

	// Follows: list -> follows section.
	if b, ok := buckets["follows"]; ok {
		if data, ok := b["list"]; ok {
			var list any
			if err := json.Unmarshal(data, &list); err == nil {
				_ = state.Write(state.Follows, list)
			}
		}
	}

	// Ordering: each key is a Record -> order section (slice of entries).
	if b, ok := buckets["ordering"]; ok {
		var entries []map[string]any
		for _, data := range b {
			var rec map[string]any
			if err := json.Unmarshal(data, &rec); err == nil {
				entries = append(entries, rec)
			}
		}
		if len(entries) > 0 {
			_ = state.Write(state.Order, entries)
		}
	}

	// Rooms: saved -> rooms section.
	if b, ok := buckets["rooms"]; ok {
		if data, ok := b["saved"]; ok {
			var rooms any
			if err := json.Unmarshal(data, &rooms); err == nil {
				_ = state.Write(state.Rooms, map[string]any{"items": rooms})
			}
		}
	}

	// Saved playlists (discord): list -> saved section.
	if b, ok := buckets["saved-playlists"]; ok {
		if data, ok := b["list"]; ok {
			var list any
			if err := json.Unmarshal(data, &list); err == nil {
				_ = state.Write(state.Saved, list)
			}
		}
	}

	// Settings: relay, sync, session-pass, discord, idle-reload.
	if b, ok := buckets["settings"]; ok {
		settings := map[string]any{}
		if data, ok := b["relay"]; ok {
			var relay any
			if json.Unmarshal(data, &relay) == nil {
				settings["relay"] = relay
			}
		}
		if data, ok := b["discord"]; ok {
			var discord bool
			if json.Unmarshal(data, &discord) == nil {
				settings["discord"] = discord
			}
		}
		if data, ok := b["idle-reload"]; ok {
			var idle bool
			if json.Unmarshal(data, &idle) == nil {
				settings["idleReload"] = idle
			}
		}
		if len(settings) > 0 {
			_ = state.Write(state.Settings, settings)
		}
		// Session pass -> sessions section.
		if data, ok := b["session-pass"]; ok {
			var pass string
			if json.Unmarshal(data, &pass) == nil && pass != "" {
				settings["sessionPass"] = pass
				_ = state.Write(state.Settings, settings)
			}
		}
	}

	// Spotify global session -> spotify section.
	if b, ok := buckets["spotify-global"]; ok {
		if data, ok := b["global"]; ok {
			var session map[string]any
			if err := json.Unmarshal(data, &session); err == nil {
				section := map[string]any{}
				if u, ok := session["username"]; ok {
					section["auth"] = map[string]any{"username": u, "data": session["data"]}
				}
				if dev, ok := buckets["spotify-device"]["device"]; ok {
					var d string
					if json.Unmarshal(dev, &d) == nil {
						section["device"] = d
					}
				}
				_ = state.Write(state.Spotify, section)
			}
		}
	}

	// YouTube auth -> youtube section.
	if b, ok := buckets["youtube-auth"]; ok {
		section := map[string]any{}
		if data, ok := b["auth"]; ok {
			var cookies string
			if json.Unmarshal(data, &cookies) == nil {
				section["cookies"] = cookies
			}
		}
		if data, ok := b["account"]; ok {
			var name string
			if json.Unmarshal(data, &name) == nil {
				section["account"] = name
			}
		}
		// youtube/sources -> youtube.links
		if src, ok := buckets["youtube-source"]; ok {
			links := map[string]string{}
			for k, v := range src {
				var vid string
				if json.Unmarshal(v, &vid) == nil {
					links[k] = vid
				}
			}
			if len(links) > 0 {
				section["links"] = links
			}
		}
		if len(section) > 0 {
			_ = state.Write(state.YouTube, section)
		}
	}

	// Window geometry -> window.json (already migrated by migrateWindowState, but do it here too).
	if b, ok := buckets["window"]; ok {
		if data, ok := b["state"]; ok {
			path := filepath.Join(filepath.Dir(dir), "window.json")
			_ = os.WriteFile(path, data, 0o644)
		}
	}

	if err := db.Close(); err != nil {
		log.Printf("migrate: cannot close legacy store: %v", err)
	}

	// Move the old store aside.
	if err := os.Rename(dir, dir+".badger"); err != nil {
		log.Printf("migrate: cannot move legacy store aside: %v", err)
	}

	return fmt.Sprintf("imported %d records from legacy store", imported), nil
}

// MigrateLegacyStoreIfPresent is a convenience wrapper that checks for a
// legacy store and migrates it if found. It returns (true, summary, nil) if
// a migration was performed.
func MigrateLegacyStoreIfPresent() (bool, string, error) {
	if !HasLegacyStore() {
		return false, "", nil
	}
	summary, err := MigrateLegacyStore()
	if err != nil {
		return false, summary, err
	}
	return true, summary, nil
}
