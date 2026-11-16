package reconcile

import "testing"

type record struct {
	id   string
	name string
	at   int64
}

func key(item record) string { return item.id }

func when(item record) int64 { return item.at }

// A removal and the record it deleted can carry the same moment; the removal
// wins, or an unfollow would be undone by the very entry it dropped.
func TestTieKeepsTombstoneAndDocumentMetadata(t *testing.T) {
	local := []record{{id: "a", name: "local", at: 5}}
	remote := []record{{id: "a", name: "remote", at: 5}}
	gone := map[string]int64{"a": 5}

	items, removed := Merge(local, remote, key, when, nil, gone, Newer)
	if len(items) != 0 {
		t.Fatalf("items = %v, want the tie to bury the record", items)
	}
	if removed["a"] != 5 {
		t.Fatalf("removed = %v, want the tombstone kept", removed)
	}

	items, removed = Merge(local, remote, key, when, nil, nil, Newer)
	if len(items) != 1 || items[0].name != "remote" {
		t.Fatalf("items = %v, want the document's copy on a tie", items)
	}
	if removed != nil {
		t.Fatalf("removed = %v, want nil", removed)
	}
}
