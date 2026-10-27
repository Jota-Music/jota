//go:build !android && !ios

package youtube

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// parseBinaryCookies reads WebKit's Cookies.binarycookies container into jar
// rows. Unlike Chromium's jar, the values are in the clear, so there is nothing
// to decrypt. The format has no documentation: it is a header naming the page
// sizes, then one block per page, and each block lists the offsets of its cookie
// records. Everything is little endian except the magic, the page count and the
// page sizes.
func parseBinaryCookies(data []byte) ([]jarRow, error) {
	if len(data) < 8 || string(data[:4]) != "cook" {
		return nil, errors.New("not a WebKit cookie file")
	}
	pages := int(binary.BigEndian.Uint32(data[4:8]))
	if pages <= 0 || len(data) < 8+pages*4 {
		return nil, errors.New("truncated WebKit cookie file")
	}

	var rows []jarRow
	offset := 8 + pages*4
	for i := range pages {
		size := int(binary.BigEndian.Uint32(data[8+i*4:]))
		if size <= 0 || offset+size > len(data) {
			return nil, errors.New("truncated WebKit cookie file")
		}
		rows = append(rows, pageRows(data[offset:offset+size])...)
		offset += size
	}
	return rows, nil
}

// pageRows collects the cookies of one page. A record that points outside its
// page is skipped: one broken entry must not cost the rest of the session.
func pageRows(page []byte) []jarRow {
	if len(page) < 8 {
		return nil
	}
	count := int(binary.LittleEndian.Uint32(page[4:8]))
	if count <= 0 || len(page) < 8+count*4 {
		return nil
	}

	var rows []jarRow
	for i := range count {
		at := int(binary.LittleEndian.Uint32(page[8+i*4:]))
		if at < 0 || at+32 > len(page) {
			continue
		}
		record := page[at:]
		field := func(i int) int { return int(binary.LittleEndian.Uint32(record[16+i*4:])) }
		row := jarRow{
			host:  cString(record, field(0)),
			name:  cString(record, field(1)),
			value: cString(record, field(3)),
		}
		if row.name == "" || row.value == "" || cookieDomainRank(row.host) == 0 {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func cString(record []byte, at int) string {
	if at <= 0 || at >= len(record) {
		return ""
	}
	end := bytes.IndexByte(record[at:], 0)
	if end < 0 {
		return ""
	}
	return string(record[at : at+end])
}
