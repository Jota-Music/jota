package youtube

import (
	"strings"
	"testing"
)

func TestFirstURL(t *testing.T) {
	out := "warning\nhttps://a/x\nhttps://b/y\n"
	if got := firstURL(out); got != "https://a/x" {
		t.Fatalf("got %q", got)
	}
	if got := firstURL("no urls"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestNetscape(t *testing.T) {
	out := netscape("SID=abc; SAPISID=xyz")
	if !strings.Contains(out, "SID\tabc") {
		t.Fatalf("missing SID row: %q", out)
	}
	if !strings.Contains(out, "SAPISID\txyz") {
		t.Fatalf("missing SAPISID row: %q", out)
	}
	if strings.Count(out, "\n") != 3 { // header + 2 rows
		t.Fatalf("rows = %d", strings.Count(out, "\n"))
	}
}

func TestBrief(t *testing.T) {
	in := "\nERROR: some reason\nmore stuff\n"
	got := brief(in)
	if got != "some reason" {
		t.Fatalf("got %q", got)
	}
}
