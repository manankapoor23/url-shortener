package main

import (
	"errors"
	"sync"
)

const maxAttempts = 5

var errNoFreeCode = errors.New("no free code after max attempts")

// store holds code → URL mappings in memory. They are lost on restart.
type store struct {
	mu       sync.RWMutex
	codes    map[string]string
	generate func(url string, attempt int) string
}

func newStore() *store {
	return &store{codes: make(map[string]string), generate: generateCode}
}

// save returns the code for url, creating one if needed.
// The same URL always gets the same code; collisions are retried.
func (s *store) save(url string) (string, error) {
	// Hashing touches no shared state, so the first attempt runs before the lock.
	code := s.generate(url, 0)

	// Check, retry and store must happen as one unit, or two requests could
	// both see a code as free and both write it.
	s.mu.Lock()
	defer s.mu.Unlock()

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			code = s.generate(url, attempt)
		}
		existing, taken := s.codes[code]
		if !taken {
			s.codes[code] = url
			return code, nil
		}
		if existing == url {
			return code, nil
		}
	}
	return "", errNoFreeCode
}

// lookup returns the URL for code. Many lookups can run at once.
func (s *store) lookup(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.codes[code]
	return url, ok
}
