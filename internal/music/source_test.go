package music

import "testing"

type fakeSource struct {
	calls []string
}

func (f *fakeSource) GetFullPlaylist(id string) (Playlist, error) {
	f.calls = append(f.calls, "full:"+id)
	return Playlist{}, nil
}

func (f *fakeSource) GetFullPlaylistNoCache(id string) (Playlist, error) {
	f.calls = append(f.calls, "nocache:"+id)
	return Playlist{}, nil
}

func (f *fakeSource) RevalidateFullPlaylist(id string) error {
	f.calls = append(f.calls, "revalidate:"+id)
	return nil
}

func TestCatalogRoutesByPrefix(t *testing.T) {
	spotify := &fakeSource{}
	youtube := &fakeSource{}
	c := NewCatalog(spotify, youtube)

	if _, err := c.GetFullPlaylist("spotify:playlist:abc"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetFullPlaylistNoCache("spotify:playlist:def"); err != nil {
		t.Fatal(err)
	}
	if err := c.RevalidateFullPlaylist("spotify:playlist:ghi"); err != nil {
		t.Fatal(err)
	}
	if _, err := spotify.GetFullPlaylist("spotify:playlist:jkl"); err != nil {
		t.Fatal(err)
	}

	if _, err := c.GetFullPlaylist("youtube:abc"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetFullPlaylistNoCache("youtube:def"); err != nil {
		t.Fatal(err)
	}
	if err := c.RevalidateFullPlaylist("youtube:ghi"); err != nil {
		t.Fatal(err)
	}

	wantSpotify := []string{"full:spotify:playlist:abc", "nocache:spotify:playlist:def", "revalidate:spotify:playlist:ghi", "full:spotify:playlist:jkl"}
	wantYoutube := []string{"full:youtube:abc", "nocache:youtube:def", "revalidate:youtube:ghi"}
	assertCalls(t, "spotify", spotify.calls, wantSpotify)
	assertCalls(t, "youtube", youtube.calls, wantYoutube)
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
