package music

import "testing"

type fakeSource struct {
	calls []string
}

func (f *fakeSource) GetFullPlaylist(id string) (Playlist, error) {
	f.calls = append(f.calls, "full:"+id)
	return Playlist{}, nil
}

func (f *fakeSource) RevalidateFullPlaylist(id string) error {
	f.calls = append(f.calls, "revalidate:"+id)
	return nil
}

func (f *fakeSource) GetSong(id string) (Song, error) {
	f.calls = append(f.calls, "song:"+id)
	return Song{}, nil
}

func TestCatalogRoutesByPrefix(t *testing.T) {
	spotify := &fakeSource{}
	youtube := &fakeSource{}
	local := &fakeSource{}
	c := NewCatalog(spotify, youtube, local)

	if _, err := c.GetFullPlaylist("spotify:playlist:abc"); err != nil {
		t.Fatal(err)
	}
	if err := c.RevalidateFullPlaylist("spotify:playlist:ghi"); err != nil {
		t.Fatal(err)
	}
	if _, err := spotify.GetFullPlaylist("spotify:playlist:jkl"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetSong("4uLU6hMCjMI75M1A2tKUQC"); err != nil {
		t.Fatal(err)
	}

	if _, err := c.GetFullPlaylist("youtube:abc"); err != nil {
		t.Fatal(err)
	}
	if err := c.RevalidateFullPlaylist("youtube:ghi"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetSong("youtube:abc"); err != nil {
		t.Fatal(err)
	}

	if _, err := c.GetFullPlaylist("local:xyz"); err != nil {
		t.Fatal(err)
	}
	if err := c.RevalidateFullPlaylist("local:xyz"); err != nil {
		t.Fatal(err)
	}

	wantSpotify := []string{
		"full:spotify:playlist:abc",
		"revalidate:spotify:playlist:ghi",
		"full:spotify:playlist:jkl",
		"song:4uLU6hMCjMI75M1A2tKUQC",
	}
	wantYoutube := []string{"full:youtube:abc", "revalidate:youtube:ghi", "song:youtube:abc"}
	wantLocal := []string{"full:local:xyz", "revalidate:local:xyz"}
	assertCalls(t, "spotify", spotify.calls, wantSpotify)
	assertCalls(t, "youtube", youtube.calls, wantYoutube)
	assertCalls(t, "local", local.calls, wantLocal)
}

func assertCalls(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s calls = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s calls[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}
