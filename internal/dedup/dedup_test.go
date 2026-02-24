package dedup

import (
	"testing"
	"time"
)

func TestStore_Check_NewID(t *testing.T) {
	s := New(time.Hour)
	if s.Check("evt-1") {
		t.Fatal("expected first check to return false (new)")
	}
}

func TestStore_Check_Duplicate(t *testing.T) {
	s := New(time.Hour)
	s.Check("evt-1")
	if !s.Check("evt-1") {
		t.Fatal("expected second check to return true (duplicate)")
	}
}

func TestStore_Check_DifferentIDs(t *testing.T) {
	s := New(time.Hour)
	s.Check("evt-1")
	if s.Check("evt-2") {
		t.Fatal("expected different ID to return false")
	}
}

func TestStore_Expiry(t *testing.T) {
	s := &Store{
		seen: make(map[string]time.Time),
		ttl:  50 * time.Millisecond,
	}
	// Don't start background cleanup — test manually

	s.Check("evt-old")

	// Manually backdate the entry
	s.mu.Lock()
	s.seen["evt-old"] = time.Now().Add(-100 * time.Millisecond)
	s.mu.Unlock()

	// Manual cleanup
	s.mu.Lock()
	cutoff := time.Now().Add(-s.ttl)
	for id, ts := range s.seen {
		if ts.Before(cutoff) {
			delete(s.seen, id)
		}
	}
	s.mu.Unlock()

	// Should be treated as new after expiry
	if s.Check("evt-old") {
		t.Fatal("expected expired ID to return false (new again)")
	}
}

func TestStore_EmptyID(t *testing.T) {
	s := New(time.Hour)
	if s.Check("") {
		t.Fatal("expected empty string first check to return false")
	}
	if !s.Check("") {
		t.Fatal("expected empty string second check to return true")
	}
}
