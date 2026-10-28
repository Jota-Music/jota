package app

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func selfRSS() int {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if fields := strings.Fields(line); len(fields) >= 2 && fields[0] == "VmRSS:" {
			kb, _ := strconv.Atoi(fields[1])
			return kb / 1024
		}
	}
	return 0
}

// RamMB must at least include the calling process, so a broken /proc walker
// (wrong indices, failing reads) fails this check on Linux.
func TestRamMB(t *testing.T) {
	self := selfRSS()
	if self == 0 {
		if _, err := os.ReadDir("/proc"); err != nil {
			t.Skip("no readable /proc")
		}
		t.Fatalf("could not read self RSS")
	}
	a := &App{}
	// The two reads are milliseconds apart and the collector can hand pages back
	// in between, so a strict comparison fails on a busy run. The margin still
	// catches a walker that answers nothing.
	if got := a.RamMB(); got < self*9/10 {
		t.Fatalf("RamMB() = %d MB, expected at least 90%% of self %d MB", got, self)
	}
}
