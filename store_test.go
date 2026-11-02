package main

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestSaveSameURLSameCode(t *testing.T) {
	s := newStore()
	a, _ := s.save("https://example.com")
	b, _ := s.save("https://example.com")
	if a != b {
		t.Fatalf("same URL got different codes: %q vs %q", a, b)
	}
}

func TestSaveRetriesOnCollision(t *testing.T) {
	s := newStore()
	// Every URL hashes to the same code per attempt, forcing collisions.
	s.generate = func(url string, attempt int) string { return fmt.Sprintf("code%03d", attempt) }

	first, _ := s.save("https://a.com")
	second, err := s.save("https://b.com")
	if err != nil || second == first {
		t.Fatalf("expected b.com to retry onto a new code, got %q, %v", second, err)
	}

	// Resubmitting b.com walks the same retry sequence and finds its code.
	again, _ := s.save("https://b.com")
	if again != second {
		t.Fatalf("resubmitted URL got %q, want existing %q", again, second)
	}
	if url, _ := s.lookup(first); url != "https://a.com" {
		t.Fatalf("collision overwrote a.com: lookup(%q) = %q", first, url)
	}
}

func TestSaveGivesUpAfterMaxAttempts(t *testing.T) {
	s := newStore()
	s.generate = func(url string, attempt int) string { return "same000" }

	s.save("https://a.com")
	if _, err := s.save("https://b.com"); !errors.Is(err, errNoFreeCode) {
		t.Fatalf("got %v, want errNoFreeCode", err)
	}
}

// Run with -race: concurrent saves and lookups must not race on the map.
func TestStoreConcurrentAccess(t *testing.T) {
	s := newStore()
	var wg sync.WaitGroup
	codes := make([]string, 100)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code, err := s.save(fmt.Sprintf("https://example.com/%d", i))
			if err != nil {
				t.Error(err)
				return
			}
			codes[i] = code
			s.lookup(code)
		}(i)
	}
	wg.Wait()

	for i, code := range codes {
		want := fmt.Sprintf("https://example.com/%d", i)
		if got, ok := s.lookup(code); !ok || got != want {
			t.Errorf("lookup(%q) = %q, want %q", code, got, want)
		}
	}
}
