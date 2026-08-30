package pcruntime

import (
	"fmt"
	"sync"
)

// boundedBuffer captures at most limit bytes and remembers how many were
// produced in total. Output is bounded at the point of capture rather than
// trimmed afterwards, so a runaway command cannot exhaust memory before anyone
// notices it needed truncating.
type boundedBuffer struct {
	mu        sync.Mutex
	limit     int64
	buf       []byte
	totalSeen int64
}

func newBoundedBuffer(limit int64) *boundedBuffer {
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.totalSeen += int64(len(p))
	if remaining := b.limit - int64(len(b.buf)); remaining > 0 {
		if int64(len(p)) <= remaining {
			b.buf = append(b.buf, p...)
		} else {
			b.buf = append(b.buf, p[:remaining]...)
		}
	}
	// Always report a full write: the command is not failing, its output is
	// being bounded, and a short write would make it die with EPIPE-like errors.
	return len(p), nil
}

// String returns the captured output, with a truncation marker when output was
// dropped. The marker is part of the value so a truncated result can never be
// mistaken for a complete one downstream.
func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.totalSeen <= int64(len(b.buf)) {
		return string(b.buf)
	}
	return string(b.buf) + truncationMarker(b.totalSeen, int64(len(b.buf)))
}

// Truncated reports whether output was dropped.
func (b *boundedBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.totalSeen > int64(len(b.buf))
}

// TotalBytes is how much the command actually produced, truncation included.
func (b *boundedBuffer) TotalBytes() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.totalSeen
}

func truncationMarker(total, kept int64) string {
	return fmt.Sprintf(
		"\n[PocketClaw runtime: output truncated, kept %d of %d bytes]", kept, total,
	)
}
