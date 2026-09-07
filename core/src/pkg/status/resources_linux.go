//go:build linux

package status

import (
	"os"
	"strconv"
	"strings"
)

// userHZ is the kernel's clock tick rate, which /proc/self/stat reports CPU
// time in. Go cannot call sysconf(_SC_CLK_TCK), and every Linux and Android
// target PocketClaw ships to uses 100.
const userHZ = 100.0

// readResources reports the gateway process's own resident memory and
// cumulative CPU time.
//
// Both come from the process's own /proc entry. Reading /proc/self never
// crosses a UID boundary, so Android's hidepid mount, which hides other
// applications' processes, does not apply. Each call is two small reads of
// kernel-generated text and no sampling: a failed or unparsable read reports
// zero for that value, which the UI renders as unavailable rather than as a
// real measurement of zero.
func readResources() Resources {
	return Resources{
		MemoryRSSBytes: readSelfRSS(),
		CPUSeconds:     readSelfCPUSeconds(),
	}
}

// readSelfRSS returns resident set size in bytes from /proc/self/statm, whose
// second field is the resident page count.
func readSelfRSS() uint64 {
	raw, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	pageSize := os.Getpagesize()
	if pageSize <= 0 {
		return 0
	}
	return pages * uint64(pageSize)
}

// readSelfCPUSeconds returns utime plus stime from /proc/self/stat, converted
// from clock ticks to seconds.
func readSelfCPUSeconds() float64 {
	raw, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	utime, stime, ok := parseSelfStatCPUTicks(string(raw))
	if !ok {
		return 0
	}
	return (float64(utime) + float64(stime)) / userHZ
}

// parseSelfStatCPUTicks extracts the utime and stime tick counts from one
// /proc/<pid>/stat line.
//
// The second field is the executable name in parentheses and may itself
// contain spaces and parentheses, so the fields are counted from after its
// closing parenthesis rather than by splitting the whole line. Counting from
// there, utime is field 12 and stime field 13.
func parseSelfStatCPUTicks(line string) (utime, stime uint64, ok bool) {
	close := strings.LastIndex(line, ")")
	if close < 0 || close+1 >= len(line) {
		return 0, 0, false
	}
	fields := strings.Fields(line[close+1:])
	if len(fields) < 13 {
		return 0, 0, false
	}
	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	stime, err = strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return utime, stime, true
}
