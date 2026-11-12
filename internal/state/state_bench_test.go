package state

import (
	"testing"
)

// BenchmarkWrite measures the cost of storing a section: one marshal plus one
// atomic rename. The writer thread amortises this across a burst, but the
// single write is what the exit flush pays.
func BenchmarkWrite(b *testing.B) {
	Reset()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Write(Playlists, map[string]int{"n": i}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRead measures the cost of reading a section back: a lookup in the
// in-memory map plus one unmarshal.
func BenchmarkRead(b *testing.B) {
	Reset()
	_ = Write(Playlists, map[string]int{"n": 1})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Read[map[string]int](Playlists); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWriteLarge measures the cost on a library-sized section, so a
// regression in the marshal or the rename shows up instead of hiding behind a
// small payload.
func BenchmarkWriteLarge(b *testing.B) {
	Reset()
	items := make([]map[string]any, 1000)
	for i := range items {
		items[i] = map[string]any{"id": i, "name": "playlist", "songs": make([]int, 50)}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Write(Playlists, items); err != nil {
			b.Fatal(err)
		}
	}
}
