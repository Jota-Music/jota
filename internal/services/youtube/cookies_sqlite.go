//go:build !android && !darwin

package youtube

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

// readJar returns the YouTube and Google cookies of a jar and the jar format
// version that says whether the encrypted values carry the host prefix. It reads
// a copy: the browser holds the live file locked, and a query against it would
// either fail or miss whatever is still only in the write-ahead log. macOS has
// no jar worth reading this way — WebKit's store is not SQLite — so it leaves the
// driver out and keeps the binary four megabytes smaller.
func readJar(path string) ([]jarRow, int, error) {
	dir, err := os.MkdirTemp("", "jota-cookies-")
	if err != nil {
		return nil, 0, err
	}
	defer os.RemoveAll(dir)

	// Keep the base name: SQLite finds the log by appending -wal/-shm to it.
	target := filepath.Join(dir, filepath.Base(path))
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := copyFile(path+suffix, target+suffix); err != nil {
			if suffix == "" {
				return nil, 0, fmt.Errorf("copying the cookie jar: %w", err)
			}
			// No log or shared-memory file is the normal case for a cleanly
			// closed browser.
			continue
		}
	}

	db, err := sql.Open("sqlite", "file:"+target+"?mode=ro")
	if err != nil {
		return nil, 0, fmt.Errorf("opening the cookie jar: %w", err)
	}
	defer db.Close()

	if firefoxJar(db) {
		list, err := firefoxRows(db)
		return list, 0, err
	}

	var version int
	var raw string
	if err := db.QueryRow(`SELECT value FROM meta WHERE key = 'version'`).Scan(&raw); err == nil {
		version, _ = strconv.Atoi(raw)
	}

	rows, err := db.Query(`SELECT host_key, name, value, encrypted_value FROM cookies
		WHERE host_key LIKE '%youtube.com%' OR host_key LIKE '%google.com%'`)
	if err != nil {
		return nil, 0, fmt.Errorf("reading the cookie jar: %w", err)
	}
	defer rows.Close()

	var list []jarRow
	for rows.Next() {
		var row jarRow
		var value sql.NullString
		if err := rows.Scan(&row.host, &row.name, &value, &row.encrypted); err != nil {
			return nil, 0, fmt.Errorf("reading the cookie jar: %w", err)
		}
		row.value = value.String
		list = append(list, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reading the cookie jar: %w", err)
	}
	return list, version, nil
}

// firefoxJar reports whether the jar is Firefox's. Firefox keeps cookies in
// moz_cookies in the clear, so it needs no keyring and no decryption at all.
// WebKitGTK's libsoup uses the same table, which is what lets the sign-in
// window's store be read with the same code.
func firefoxJar(db *sql.DB) bool {
	var table string
	return db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'moz_cookies'`).Scan(&table) == nil
}

// firefoxRows returns the YouTube and Google cookies of a Firefox jar.
func firefoxRows(db *sql.DB) ([]jarRow, error) {
	rows, err := db.Query(`SELECT host, name, value FROM moz_cookies
		WHERE host LIKE '%youtube.com%' OR host LIKE '%google.com%'`)
	if err != nil {
		return nil, fmt.Errorf("reading the cookie jar: %w", err)
	}
	defer rows.Close()

	var list []jarRow
	for rows.Next() {
		var row jarRow
		if err := rows.Scan(&row.host, &row.name, &row.value); err != nil {
			return nil, fmt.Errorf("reading the cookie jar: %w", err)
		}
		list = append(list, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the cookie jar: %w", err)
	}
	return list, nil
}

func copyFile(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(to)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
