package ordering

import (
	"reflect"
	"testing"

	"github.com/Jota-Music/jota/internal/state"
)

func setup(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	state.Reset()
	t.Cleanup(state.Reset)
	return New()
}

func entry(t *testing.T, part Section, scope, account string) Entry {
	t.Helper()
	for _, e := range part {
		if e.Scope == scope && e.Account == account {
			return e
		}
	}
	t.Fatalf("section has no %s/%s entry: %+v", scope, account, part)
	return Entry{}
}

func TestOrderLifecycle(t *testing.T) {
	s := setup(t)

	if ids, err := s.List(Library, "alice"); err != nil || len(ids) != 0 {
		t.Fatalf("empty list = %v, %v", ids, err)
	}

	if err := s.Save(Library, "Alice", Order{"b", "a", "b", "  ", "c"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List(Library, "ALICE")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Order{"b", "a", "c"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}

	other, err := s.List(Library, "carol")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other account leaked order: %v", other)
	}
}

// The rooms shelf keeps ids in the app's naming rule, including an order
// written before it (`<relay>|<code>`).
func TestRoomsOrderFollowsTheNamingRule(t *testing.T) {
	s := setup(t)

	if err := s.Save(Rooms, "", Order{"relay.example.com|party", "local:room:other.example.com|jam"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List(Rooms, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := Order{"local:room:relay.example.com|party", "local:room:other.example.com|jam"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}

	// A document written before the rule is read and written back composed.
	legacy := Section{{Scope: Rooms, Ids: []string{"relay.example.com|party"}, At: 9}}
	merged, applied, err := s.MergeSection(legacy, Restore, true)
	if err != nil || applied == 0 {
		t.Fatalf("merge = %v, %d, %v", merged, applied, err)
	}
	if id := entry(t, merged, Rooms, "").Ids[0]; id != "local:room:relay.example.com|party" {
		t.Fatalf("merged id = %q", id)
	}
	if got, _ := s.List(Rooms, ""); len(got) != 1 || got[0] != "local:room:relay.example.com|party" {
		t.Fatalf("stored = %v", got)
	}
}

// The following shelf names a follow by the app rule too, including an order
// written before it.
func TestFollowingOrderFollowsTheNamingRule(t *testing.T) {
	s := setup(t)

	if err := s.Save(Following, "", Order{
		"artist:4uLU6hMCjMI75M1A2tKUQC",
		"youtube:UCY4YovmJwl4_BivQ-LkzU2A",
		"aokihiro",
		"spotify:artist:abc",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List(Following, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := Order{
		"spotify:artist:4uLU6hMCjMI75M1A2tKUQC",
		"youtube:channel:UCY4YovmJwl4_BivQ-LkzU2A",
		"spotify:user:aokihiro",
		"spotify:artist:abc",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}
}

func TestScopesAreSeparate(t *testing.T) {
	s := setup(t)

	if err := s.Save(Library, "alice", Order{"a", "b"}); err != nil {
		t.Fatalf("save library: %v", err)
	}
	if err := s.Save("following", "alice", Order{"spotify:artist:b", "spotify:artist:a"}); err != nil {
		t.Fatalf("save following: %v", err)
	}

	library, _ := s.List(Library, "alice")
	if want := (Order{"a", "b"}); !reflect.DeepEqual(library, want) {
		t.Fatalf("library = %v, want %v", library, want)
	}
	following, _ := s.List("following", "alice")
	if want := (Order{"spotify:artist:b", "spotify:artist:a"}); !reflect.DeepEqual(following, want) {
		t.Fatalf("following = %v, want %v", following, want)
	}
}

func TestOrderWithoutAccount(t *testing.T) {
	s := setup(t)

	if err := s.Save("following", "", Order{"spotify:user:a", "spotify:user:b"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List("following", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Order{"spotify:user:a", "spotify:user:b"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("following = %v, want %v", got, want)
	}
	part, err := s.Section()
	if err != nil {
		t.Fatalf("section: %v", err)
	}
	rec := entry(t, part, "following", "")
	if rec.Account != "" || !reflect.DeepEqual(Order(rec.Ids), Order{"spotify:user:a", "spotify:user:b"}) {
		t.Fatalf("section = %+v, want the shelf without an account", rec)
	}
}

func TestMergeSectionKeepsTheNewestList(t *testing.T) {
	s := setup(t)
	if err := s.Save(Library, "alice", Order{"a", "b"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	local, _ := s.Section()
	at := entry(t, local, Library, "alice").At

	remote := Section{
		{Scope: Library, Account: "alice", Ids: []string{"b", "a"}, At: at + 1},
		{Scope: "following", Account: "alice", Ids: []string{"c"}, At: 1},
	}
	out, applied, err := s.MergeSection(remote, Newer, true)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if applied != 2 {
		t.Fatalf("applied = %d, want 2", applied)
	}
	if got := Order(entry(t, out, Library, "alice").Ids); !reflect.DeepEqual(got, Order{"b", "a"}) {
		t.Fatalf("library = %v, want the newer list from the file", got)
	}
	if _, _, err := s.MergeSection(Section{
		{Scope: Library, Account: "alice", Ids: []string{"a", "b"}, At: at - 1},
	}, Newer, true); err != nil {
		t.Fatalf("merge older: %v", err)
	}
	if got, _ := s.List(Library, "alice"); !reflect.DeepEqual(got, Order{"b", "a"}) {
		t.Fatalf("library = %v, want the newer local list kept", got)
	}

	// An explicit restore takes the document's list whatever its age, because the
	// user asked for that document and not for its age.
	if _, _, err := s.MergeSection(Section{
		{Scope: Library, Account: "alice", Ids: []string{"a", "b"}, At: at - 1},
	}, Restore, true); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got, _ := s.List(Library, "alice"); !reflect.DeepEqual(got, Order{"a", "b"}) {
		t.Fatalf("library = %v, want the document's list after a restore", got)
	}
}

func TestMergeSectionWithoutApplyLeavesStore(t *testing.T) {
	s := setup(t)
	if err := s.Save(Library, "alice", Order{"a"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	remote := Section{{Scope: Library, Account: "bob", Ids: []string{"x"}, At: 5}}
	out, applied, err := s.MergeSection(remote, Newer, false)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if applied != 0 || len(out) != 2 {
		t.Fatalf("applied = %d, union = %v, want 0 and both shelves", applied, out)
	}
	if got, _ := s.List(Library, "bob"); len(got) != 0 {
		t.Fatalf("store = %v, want the union only in the answer", got)
	}
}
