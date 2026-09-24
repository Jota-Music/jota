package update

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		tag  string
		want [3]int
		ok   bool
	}{
		{"v1.2.3", [3]int{1, 2, 3}, true},
		{"0.0.0", [3]int{0, 0, 0}, true},
		{"v2.0.0-rc1", [3]int{2, 0, 0}, true},
		{"1.2.3+build", [3]int{1, 2, 3}, true},
		{"dev", [3]int{}, false},
		{"1.2", [3]int{}, false},
		{"1.2.x", [3]int{}, false},
	}
	for _, c := range cases {
		got, ok := parse(c.tag)
		if ok != c.ok || got != c.want {
			t.Errorf("parse(%q) = %v, %v; want %v, %v", c.tag, got, ok, c.want, c.ok)
		}
	}
}

func TestNewer(t *testing.T) {
	cases := []struct {
		latest  string
		current string
		want    bool
	}{
		{"0.2.0", "0.1.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"1.0.0", "0.9.9", true},
		{"0.2.0", "dev", false},
	}
	for _, c := range cases {
		if got := newer(c.latest, c.current); got != c.want {
			t.Errorf("newer(%q, %q) = %v; want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestPick(t *testing.T) {
	releases := []release{
		{TagName: "v0.1.0"},
		{TagName: "v0.3.0", Draft: true},
		{TagName: "not-a-version"},
		{TagName: "v0.2.0"},
	}
	got := pick(releases)
	if got == nil || got.TagName != "v0.2.0" {
		t.Fatalf("pick = %+v; want v0.2.0", got)
	}
	if pick(nil) != nil {
		t.Fatal("pick(nil) should be nil")
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		ch   Channel
		name string
		want bool
	}{
		{MacOS, "jota-macos.zip", true},
		{MacOS, "jota.dmg", false},
		{Windows, "jota-windows-amd64.exe", true},
		{Windows, "jota-amd64-installer.exe", false},
		{Windows, "jota-macos.zip", false},
		{Android, "jota.apk", true},
		{Android, "jota.deb", false},
		{External, "jota.deb", false},
	}
	for _, c := range cases {
		if got := matches(c.ch, c.name); got != c.want {
			t.Errorf("matches(%s, %q) = %v; want %v", c.ch, c.name, got, c.want)
		}
	}
}

func TestContainsArch(t *testing.T) {
	if !containsArch("jota-x86_64.appimage", "amd64") {
		t.Error("x86_64 should match amd64")
	}
	if !containsArch("jota-aarch64.appimage", "arm64") {
		t.Error("aarch64 should match arm64")
	}
	if containsArch("jota-x86_64.appimage", "arm64") {
		t.Error("x86_64 must not match arm64")
	}
}

func TestParseSums(t *testing.T) {
	body := "abc123  jota.deb\ndef456  *jota-x86_64.AppImage\nghi789  ./jota-macos.zip\n"
	got, err := parseSums(body, "jota-x86_64.AppImage")
	if err != nil || got != "def456" {
		t.Fatalf("parseSums = %q, %v; want def456", got, err)
	}
	got, err = parseSums(body, "jota-macos.zip")
	if err != nil || got != "ghi789" {
		t.Fatalf("parseSums path-prefixed = %q, %v; want ghi789", got, err)
	}
	if _, err := parseSums(body, "missing"); err == nil {
		t.Fatal("expected error for a missing entry")
	}
}
