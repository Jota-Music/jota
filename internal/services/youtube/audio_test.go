package youtube

import "testing"

func TestBestAudioPrefersMP4(t *testing.T) {
	cases := []struct {
		name    string
		formats []format
		want    int
		ok      bool
	}{
		{
			name: "prefers mp4 over higher webm itag",
			formats: []format{
				{Itag: 251, MimeType: `audio/webm; codecs="opus"`, URL: "webm"},
				{Itag: 140, MimeType: `audio/mp4; codecs="mp4a.40.2"`, URL: "mp4"},
			},
			want: 140,
			ok:   true,
		},
		{
			name:    "falls back to webm when no mp4",
			formats: []format{{Itag: 251, MimeType: `audio/webm; codecs="opus"`, URL: "webm"}},
			want:    251,
			ok:      true,
		},
		{
			name: "highest mp4 wins within tier",
			formats: []format{
				{Itag: 139, MimeType: "audio/mp4", URL: "a"},
				{Itag: 140, MimeType: "audio/mp4", URL: "b"},
			},
			want: 140,
			ok:   true,
		},
		{
			name: "ignores non-audio and empty url",
			formats: []format{
				{Itag: 137, MimeType: "video/mp4", URL: "v"},
				{Itag: 140, MimeType: "audio/mp4", URL: ""},
			},
			ok: false,
		},
		{name: "empty", formats: nil, ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := bestAudio(c.formats)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if ok && got.Itag != c.want {
				t.Fatalf("itag = %d, want %d", got.Itag, c.want)
			}
		})
	}
}
