package cache

import (
	"testing"
)

// BenchmarkPut measures the cost of storing one entity: marshal plus an atomic
// rename. The trim is not on this path.
func BenchmarkPut(b *testing.B) {
	setupBench(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Put("playlists/one", "a playlist"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGet measures the cost of reading one entity back: a file read plus
// one unmarshal, plus the expiry check.
func BenchmarkGet(b *testing.B) {
	setupBench(b)
	_ = Put("playlists/one", "a playlist")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := Get[string]("playlists/one"); !ok {
			b.Fatal("missing")
		}
	}
}

func setupBench(b *testing.B) {
	b.Helper()
	dir := b.TempDir()
	b.Setenv("HOME", dir)
	b.Setenv("XDG_CONFIG_HOME", dir)
	b.Setenv("APPDATA", dir)
}
