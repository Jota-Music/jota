//go:build !android && !ios

package youtube

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildCookieFile lays out one WebKit page holding the given cookies, which is
// the shape parseBinaryCookies has to read back.
func buildCookieFile(cookies []jarRow) []byte {
	var records bytes.Buffer
	offsets := make([]int, len(cookies))
	for i, cookie := range cookies {
		offsets[i] = records.Len()
		record := make([]byte, 56)

		var body bytes.Buffer
		fields := []string{cookie.host, cookie.name, "/", cookie.value}
		at := make([]int, len(fields))
		for j, field := range fields {
			at[j] = 56 + body.Len()
			body.WriteString(field)
			body.WriteByte(0)
		}
		binary.LittleEndian.PutUint32(record[0:4], uint32(56+body.Len()))
		for j, off := range at {
			binary.LittleEndian.PutUint32(record[16+j*4:], uint32(off))
		}
		records.Write(record)
		records.Write(body.Bytes())
	}

	page := make([]byte, 8)
	binary.LittleEndian.PutUint32(page[4:8], uint32(len(cookies)))
	for _, off := range offsets {
		var slot [4]byte
		binary.LittleEndian.PutUint32(slot[:], uint32(8+len(offsets)*4+off))
		page = append(page, slot[:]...)
	}
	page = append(page, records.Bytes()...)

	file := make([]byte, 12)
	copy(file[0:4], "cook")
	binary.BigEndian.PutUint32(file[4:8], 1)
	binary.BigEndian.PutUint32(file[8:12], uint32(len(page)))
	return append(file, page...)
}

func TestParseBinaryCookies(t *testing.T) {
	file := buildCookieFile([]jarRow{
		{host: ".google.com", name: "SID", value: "google"},
		{host: ".youtube.com", name: "SID", value: "youtube"},
		{host: ".youtube.com", name: "__Secure-3PSID", value: "session"},
		{host: ".example.com", name: "other", value: "dropped"},
	})

	rows, err := parseBinaryCookies(file)
	if err != nil {
		t.Fatalf("parseBinaryCookies = %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	header, _ := cookieHeader(rows, nil)
	if value := cookieValue(header, "SID"); value != "youtube" {
		t.Fatalf("SID = %q, want the youtube.com copy", value)
	}
	if sessionValue(header) == "" {
		t.Fatal("header lost the private session cookie, so SessionFull would answer an incomplete session")
	}
}

func TestParseBinaryCookiesRejects(t *testing.T) {
	if _, err := parseBinaryCookies([]byte("nope")); err == nil {
		t.Fatal("a file that is not a cookie jar must fail")
	}
	if _, err := parseBinaryCookies(buildCookieFile(nil)[:14]); err == nil {
		t.Fatal("a truncated page must fail")
	}
}
