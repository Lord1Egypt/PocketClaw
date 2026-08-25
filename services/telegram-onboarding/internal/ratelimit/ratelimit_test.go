package ratelimit

import (
	"testing"
	"time"
)

func TestAllowsBurstThenBlocks(t *testing.T) {
	l := New(3, 1)
	now := time.Now()
	l.SetClock(func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("request %d within the burst was blocked", i+1)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("a request beyond the burst was allowed")
	}
}

func TestRefillsOverTime(t *testing.T) {
	l := New(2, 1)
	now := time.Now()
	l.SetClock(func() time.Time { return now })

	l.Allow("ip")
	l.Allow("ip")
	if l.Allow("ip") {
		t.Fatal("bucket was not empty")
	}
	now = now.Add(2 * time.Second)
	if !l.Allow("ip") {
		t.Fatal("bucket did not refill after 2 seconds at 1/s")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l := New(1, 1)
	if !l.Allow("a") {
		t.Fatal("first key blocked")
	}
	if !l.Allow("b") {
		t.Fatal("a second key was blocked by the first key's usage")
	}
	if l.Allow("a") {
		t.Fatal("first key was refilled by another key's request")
	}
}

func TestIdleBucketsAreReclaimed(t *testing.T) {
	l := New(1, 1)
	now := time.Now()
	l.SetClock(func() time.Time { return now })
	l.Allow("ip")
	now = now.Add(11 * time.Minute)
	l.Allow("other")

	l.mu.Lock()
	_, stillThere := l.buckets["ip"]
	l.mu.Unlock()
	if stillThere {
		t.Fatal("an idle bucket was not reclaimed")
	}
}
