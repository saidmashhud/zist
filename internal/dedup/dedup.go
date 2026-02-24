// Package dedup provides an in-memory TTL deduplication store for webhook event IDs.
package dedup

import (
	"sync"
	"time"
)

// Store tracks seen event IDs with a configurable TTL.
// After the TTL expires, entries are automatically cleaned up.
type Store struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

// New creates a Store that remembers event IDs for the given TTL.
// A background goroutine purges expired entries every minute.
func New(ttl time.Duration) *Store {
	s := &Store{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
	go s.cleanup()
	return s
}

// Check returns true if eventID has already been seen (duplicate).
// If it's new, the ID is recorded and false is returned.
func (s *Store) Check(eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.seen[eventID]; ok {
		return true
	}
	s.seen[eventID] = time.Now()
	return false
}

func (s *Store) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-s.ttl)
		for id, ts := range s.seen {
			if ts.Before(cutoff) {
				delete(s.seen, id)
			}
		}
		s.mu.Unlock()
	}
}
