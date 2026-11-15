// Package legacy reads the Badger store the app kept before the state
// document, without the Badger library.
//
// The store is read directly: the manifest (protobuf), the table index
// (flatbuffers) and the blocks (zstd). The values that lived in the value log
// are the cached ones, which the sources answer again, so they are not read.
package legacy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

// Store is what the legacy store holds, grouped by bucket.
type Store map[string]map[string][]byte

// The bits of the meta byte of an entry.
const (
	deleted      byte = 1 << 0 // the key was deleted
	valuePointer byte = 1 << 1 // the value lives in the value log
)

// vlogHeaderSize is what the value log writes before its entries.
const vlogHeaderSize = 20

// magicVersion is the version of the store format this reader understands.
// A store written by another version is refused, not guessed.
const magicVersion = 8

// crc32c is the checksum the store carries.
var crc32c = crc32.MakeTable(crc32.Castagnoli)

// Read reads the live values of the store at dir, grouped by bucket. The
// values that the value log holds are not read: they are the cached ones.
// The number of those is returned so the caller can say what was left out.
func Read(dir string) (Store, int, error) {
	tables, err := readManifest(filepath.Join(dir, "MANIFEST"))
	if err != nil {
		return nil, 0, err
	}
	latest := entries{}
	for _, t := range tables {
		found, err := readTable(filepath.Join(dir, fmt.Sprintf("%06d.sst", t.id)), t.compression)
		if err != nil {
			return nil, 0, err
		}
		for _, e := range found {
			latest.keep(e)
		}
	}
	for _, path := range walFiles(dir) {
		found, err := readWAL(path)
		if err != nil {
			return nil, 0, err
		}
		for _, e := range found {
			latest.keep(e)
		}
	}

	store := Store{}
	skipped := 0
	for _, e := range latest {
		if e.meta&valuePointer != 0 {
			skipped++
			continue
		}
		if e.meta&deleted != 0 {
			continue
		}
		if e.expired(time.Now()) {
			continue
		}
		bucket, key, ok := strings.Cut(e.key, ":")
		if !ok {
			continue
		}
		if store[bucket] == nil {
			store[bucket] = map[string][]byte{}
		}
		store[bucket][key] = e.value
	}
	return store, skipped, nil
}

// entry is one value with the version it was written at.
type entry struct {
	key       string
	value     []byte
	meta      byte
	expiresAt uint64
	version   uint64
}

// entries keeps the newest version of every key.
type entries map[string]entry

func (e entries) keep(candidate entry) {
	old, ok := e[candidate.key]
	if ok && old.version >= candidate.version {
		return
	}
	e[candidate.key] = candidate
}

// expired reports whether the TTL the entry carried has passed. An entry
// without one never expires.
func (e entry) expired(now time.Time) bool {
	return e.expiresAt != 0 && e.expiresAt <= uint64(now.Unix())
}

// --- the manifest ---

// table is a table of the LSM: the file and the compression it was written
// with.
type table struct {
	id          uint64
	compression uint32
}

// readManifest reads the tables the store is made of.
func readManifest(path string) ([]table, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 8 || string(raw[:4]) != "Bdgr" {
		return nil, errors.New("legacy: not a badger store")
	}
	if version := binary.BigEndian.Uint16(raw[6:8]); version != magicVersion {
		return nil, fmt.Errorf("legacy: the store is version %d and this reader knows %d", version, magicVersion)
	}
	live := map[uint64]table{}
	pos := 8
	for pos+8 <= len(raw) {
		length := int(binary.BigEndian.Uint32(raw[pos:]))
		checksum := binary.BigEndian.Uint32(raw[pos+4:])
		pos += 8
		if length < 0 || pos+length > len(raw) {
			break
		}
		if length == 0 {
			continue
		}
		body := raw[pos : pos+length]
		if crc32.Checksum(body, crc32c) != checksum {
			return nil, errors.New("legacy: the store is damaged (the manifest does not match its checksum)")
		}
		for _, c := range parseChanges(body) {
			if c.op == 0 {
				live[c.id] = table{id: c.id, compression: c.compression}
			} else {
				delete(live, c.id)
			}
		}
		pos += length
	}
	tables := make([]table, 0, len(live))
	for _, t := range live {
		tables = append(tables, t)
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].id < tables[j].id })
	return tables, nil
}

// change is one change of the manifest: a table that was created or deleted.
type change struct {
	id          uint64
	op          int
	compression uint32
}

// parseChanges reads the protobuf ManifestChangeSet: field 1 holds the
// changes.
func parseChanges(buf []byte) []change {
	var out []change
	for len(buf) > 0 {
		tag, n := binary.Uvarint(buf)
		if n <= 0 {
			return out
		}
		buf = buf[n:]
		field, wire := tag>>3, tag&7
		switch wire {
		case 0:
			_, n := binary.Uvarint(buf)
			if n <= 0 {
				return out
			}
			buf = buf[n:]
		case 2:
			length, n := binary.Uvarint(buf)
			if n <= 0 || uint64(len(buf)-n) < length {
				return out
			}
			buf = buf[n:]
			if field == 1 {
				out = append(out, parseChange(buf[:length]))
			}
			buf = buf[length:]
		case 5:
			if len(buf) < 4 {
				return out
			}
			buf = buf[4:]
		case 1:
			if len(buf) < 8 {
				return out
			}
			buf = buf[8:]
		default:
			return out
		}
	}
	return out
}

// parseChange reads the protobuf ManifestChange: 1 the id, 2 the operation
// (0 creates, 1 deletes) and 6 the compression.
func parseChange(buf []byte) change {
	var c change
	for len(buf) > 0 {
		tag, n := binary.Uvarint(buf)
		if n <= 0 {
			return c
		}
		buf = buf[n:]
		field, wire := tag>>3, tag&7
		if wire != 0 {
			switch wire {
			case 2:
				length, n := binary.Uvarint(buf)
				if n <= 0 || uint64(len(buf)-n) < length {
					return c
				}
				buf = buf[n+int(length):]
			case 5:
				if len(buf) < 4 {
					return c
				}
				buf = buf[4:]
			case 1:
				if len(buf) < 8 {
					return c
				}
				buf = buf[8:]
			default:
				return c
			}
			continue
		}
		value, n := binary.Uvarint(buf)
		if n <= 0 {
			return c
		}
		buf = buf[n:]
		switch field {
		case 1:
			c.id = value
		case 2:
			c.op = int(value)
		case 6:
			c.compression = uint32(value)
		}
	}
	return c
}

// --- the tables ---

// readTable reads every entry of one table.
func readTable(path string, compression uint32) ([]entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 16 {
		return nil, errors.New("legacy: a table is too short to hold an index")
	}
	// The table ends with the index, the checksum of the index and the two
	// lengths that say where the index starts.
	checksumLen := int(binary.BigEndian.Uint32(raw[len(raw)-4:]))
	if checksumLen < 0 || checksumLen > len(raw)-8 {
		return nil, errors.New("legacy: a table is damaged (the index does not fit)")
	}
	indexLen := int(binary.BigEndian.Uint32(raw[len(raw)-8-checksumLen:]))
	if indexLen < 0 || indexLen > len(raw)-8-checksumLen {
		return nil, errors.New("legacy: a table is damaged (the index does not fit)")
	}
	start := len(raw) - 8 - checksumLen - indexLen
	index := raw[start : len(raw)-8-checksumLen]
	if err := verifyChecksum(index, raw[len(raw)-4-checksumLen:len(raw)-4]); err != nil {
		return nil, err
	}
	offsets, err := parseIndex(index)
	if err != nil {
		return nil, err
	}

	var decoder *zstd.Decoder
	if compression == 2 {
		decoder, err = zstd.NewReader(nil)
		if err != nil {
			return nil, err
		}
		defer decoder.Close()
	}
	var out []entry
	for _, o := range offsets {
		if int(o.offset)+int(o.length) > len(raw) {
			return nil, errors.New("legacy: a table is damaged (a block does not fit)")
		}
		block := raw[o.offset : o.offset+o.length]
		if decoder != nil {
			plain, err := decoder.DecodeAll(block, nil)
			if err != nil {
				return nil, fmt.Errorf("legacy: a table is damaged (a block does not decode): %w", err)
			}
			block = plain
		}
		entries, err := parseBlock(block, o.key)
		if err != nil {
			return nil, err
		}
		out = append(out, entries...)
	}
	return out, nil
}

// parseBlock reads the entries of one block: the offsets, and then every
// entry's header, diff key and value.
func parseBlock(data []byte, baseKey []byte) ([]entry, error) {
	if len(data) < 8 {
		return nil, errors.New("legacy: a block is too short")
	}
	readPos := len(data) - 4
	checksumLen := int(binary.BigEndian.Uint32(data[readPos:]))
	if checksumLen < 0 || checksumLen > readPos {
		return nil, errors.New("legacy: a block is damaged (its checksum does not fit)")
	}
	readPos -= checksumLen
	if err := verifyChecksum(data[:readPos], data[readPos:readPos+checksumLen]); err != nil {
		return nil, err
	}
	readPos -= 4
	count := int(binary.BigEndian.Uint32(data[readPos:]))
	start := readPos - count*4
	if start < 0 {
		return nil, errors.New("legacy: a block is damaged (its offsets do not fit)")
	}
	offsets := data[start:readPos]

	var out []entry
	for i := 0; i < count; i++ {
		offset := int(binary.LittleEndian.Uint32(offsets[i*4:]))
		limit := start
		if i+1 < count {
			limit = int(binary.LittleEndian.Uint32(offsets[(i+1)*4:]))
		}
		if offset+4 > limit || limit > len(data) {
			return nil, errors.New("legacy: a block is damaged (an entry does not fit)")
		}
		overlap := int(binary.LittleEndian.Uint16(data[offset:]))
		diff := int(binary.LittleEndian.Uint16(data[offset+2:]))
		pos := offset + 4
		if pos+diff > limit {
			return nil, errors.New("legacy: a block is damaged (a key does not fit)")
		}
		diffKey := data[pos : pos+diff]
		pos += diff
		if pos+2 > limit {
			return nil, errors.New("legacy: a block is damaged (a value does not fit)")
		}
		meta := data[pos]
		pos += 2
		expires, n := binary.Uvarint(data[pos:])
		pos += n
		if pos > limit {
			return nil, errors.New("legacy: a block is damaged (a value does not fit)")
		}
		key := make([]byte, 0, len(baseKey)+len(diffKey))
		key = append(key, baseKey[:overlap]...)
		key = append(key, diffKey...)
		if len(key) < 8 {
			return nil, errors.New("legacy: a block is damaged (a key is too short)")
		}
		out = append(out, entry{
			key:       string(key[:len(key)-8]),
			value:     data[pos:limit],
			meta:      meta,
			expiresAt: expires,
			version:   ^binary.BigEndian.Uint64(key[len(key)-8:]),
		})
	}
	return out, nil
}

// verifyChecksum checks the data against the checksum the store carries. The
// checksum is a protobuf: the algorithm and the sum.
func verifyChecksum(data []byte, checksum []byte) error {
	algo, sum, err := parseChecksum(checksum)
	if err != nil {
		return err
	}
	if algo != 0 {
		return fmt.Errorf("legacy: the store uses a checksum this reader does not know (%d)", algo)
	}
	if uint64(crc32.Checksum(data, crc32c)) != sum {
		return errors.New("legacy: the store is damaged (a block does not match its checksum)")
	}
	return nil
}

// parseChecksum reads the protobuf Checksum: 1 the algorithm, 2 the sum.
func parseChecksum(buf []byte) (int, uint64, error) {
	var algo int
	var sum uint64
	for len(buf) > 0 {
		tag, n := binary.Uvarint(buf)
		if n <= 0 {
			return 0, 0, errors.New("legacy: the store is damaged (a checksum does not decode)")
		}
		buf = buf[n:]
		field, wire := tag>>3, tag&7
		if wire != 0 {
			switch wire {
			case 2:
				length, n := binary.Uvarint(buf)
				if n <= 0 || int(length) > len(buf) {
					return 0, 0, errors.New("legacy: the store is damaged (a checksum does not decode)")
				}
				buf = buf[n+int(length):]
			case 5:
				if len(buf) < 4 {
					return 0, 0, errors.New("legacy: the store is damaged (a checksum is cut short)")
				}
				buf = buf[4:]
			case 1:
				if len(buf) < 8 {
					return 0, 0, errors.New("legacy: the store is damaged (a checksum is cut short)")
				}
				buf = buf[8:]
			default:
				return 0, 0, errors.New("legacy: the store is damaged (a checksum does not decode)")
			}
			continue
		}
		value, n := binary.Uvarint(buf)
		if n <= 0 {
			return 0, 0, errors.New("legacy: the store is damaged (a checksum does not decode)")
		}
		buf = buf[n:]
		switch field {
		case 1:
			algo = int(value)
		case 2:
			sum = value
		}
	}
	return algo, sum, nil
}

// --- the table index ---

// blockOffset is where a block lives and the key its entries share.
type blockOffset struct {
	key    []byte
	offset uint32
	length uint32
}

// parseIndex reads the flatbuffers index of a table. An index it cannot read is
// an error and not an empty table: a table that is silently dropped from the
// import is a half import, and the store would be moved aside as if complete.
func parseIndex(buf []byte) ([]blockOffset, error) {
	damaged := errors.New("legacy: a table is damaged (the index does not decode)")
	if len(buf) < 8 {
		return nil, damaged
	}
	root := fbTable{buf, int(binary.LittleEndian.Uint32(buf))}
	field := root.field(0)
	if field == 0 {
		// An empty table has no vector to point at, and carrying nothing from it
		// is the whole of the import it asks for.
		return nil, nil
	}
	vector := root.indirect(root.pos + field)
	if vector < 0 || vector+4 > len(buf) {
		return nil, damaged
	}
	count := int(binary.LittleEndian.Uint32(buf[vector:]))
	out := make([]blockOffset, 0, count)
	for i := 0; i < count; i++ {
		element := vector + 4 + i*4
		if element+4 > len(buf) {
			break
		}
		table := fbTable{buf, root.indirect(element)}
		var o blockOffset
		if f := table.field(0); f != 0 {
			vector := table.indirect(table.pos + f)
			if vector+4 <= len(buf) {
				length := int(binary.LittleEndian.Uint32(buf[vector:]))
				if vector+4+length <= len(buf) {
					o.key = buf[vector+4 : vector+4+length]
				}
			}
		}
		if f := table.field(1); f != 0 && table.pos+f+4 <= len(buf) {
			o.offset = binary.LittleEndian.Uint32(buf[table.pos+f:])
		}
		if f := table.field(2); f != 0 && table.pos+f+4 <= len(buf) {
			o.length = binary.LittleEndian.Uint32(buf[table.pos+f:])
		}
		out = append(out, o)
	}
	return out, nil
}

// fbTable is a flatbuffers table: a place in the buffer plus the vtable that
// describes its fields.
type fbTable struct {
	buf []byte
	pos int
}

// field returns where a field sits inside the table, or 0 when the table does
// not have it.
func (t fbTable) field(field int) int {
	if t.pos+4 > len(t.buf) {
		return 0
	}
	back := int(int32(binary.LittleEndian.Uint32(t.buf[t.pos:])))
	vtable := t.pos - back
	if vtable < 0 || vtable+4 > len(t.buf) {
		return 0
	}
	length := int(binary.LittleEndian.Uint16(t.buf[vtable:]))
	slot := 4 + field*2
	if slot+2 > length {
		return 0
	}
	return int(binary.LittleEndian.Uint16(t.buf[vtable+slot:]))
}

// indirect resolves a relative offset stored at a place.
func (t fbTable) indirect(pos int) int {
	if pos < 0 || pos+4 > len(t.buf) {
		return 0
	}
	return pos + int(binary.LittleEndian.Uint32(t.buf[pos:]))
}

// --- the memtable log ---

// walFiles are the logs of the memtables: the writes that are not in the LSM
// yet.
func walFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mem") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

// readWAL reads the entries of a memtable log: a header, the key, the value
// and a checksum, one after the other until a zeroed entry.
func readWAL(path string) ([]entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) <= vlogHeaderSize {
		return nil, nil
	}
	var out []entry
	pos := vlogHeaderSize
	for pos < len(raw) {
		if pos+1 >= len(raw) || (raw[pos] == 0 && raw[pos+1] == 0) {
			break
		}
		start := pos
		meta := raw[pos]
		pos += 2
		keyLen, n := binary.Uvarint(raw[pos:])
		pos += n
		valueLen, n := binary.Uvarint(raw[pos:])
		pos += n
		expires, n := binary.Uvarint(raw[pos:])
		pos += n
		// A delete carries no value, so an empty one is an entry and not the
		// end of the log: the writes that follow it are still the newest ones.
		rest := len(raw) - pos
		if keyLen == 0 || keyLen > uint64(rest) || valueLen > uint64(rest) ||
			keyLen+valueLen+4 > uint64(rest) {
			break
		}
		key := raw[pos : pos+int(keyLen)]
		value := raw[pos+int(keyLen) : pos+int(keyLen)+int(valueLen)]
		pos += int(keyLen) + int(valueLen)
		checksum := binary.BigEndian.Uint32(raw[pos:])
		pos += 4
		if crc32.Checksum(raw[start:pos-4], crc32c) != checksum {
			return nil, errors.New("legacy: the store is damaged (a log entry does not match its checksum)")
		}
		if len(key) < 8 {
			continue
		}
		out = append(out, entry{
			key:       string(key[:len(key)-8]),
			value:     value,
			meta:      meta,
			expiresAt: expires,
			version:   ^binary.BigEndian.Uint64(key[len(key)-8:]),
		})
	}
	return out, nil
}
