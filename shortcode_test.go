package main

import "testing"

func TestGenerateCodeIsDeterministic(t *testing.T) {
	a := generateCode("https://example.com", 0)
	b := generateCode("https://example.com", 0)
	if a != b {
		t.Fatalf("same URL gave different codes: %q vs %q", a, b)
	}
}

func TestGenerateCodeShape(t *testing.T) {
	for _, url := range []string{"https://a.co", "https://example.com/a/very/long/path?with=query"} {
		code := generateCode(url, 0)
		if !isValidCode(code) {
			t.Errorf("generateCode(%q) = %q, want 7 Base62 chars", url, code)
		}
	}
}

func TestGenerateCodeRetryChangesCode(t *testing.T) {
	if generateCode("https://example.com", 0) == generateCode("https://example.com", 1) {
		t.Fatal("attempt 1 should produce a different code than attempt 0")
	}
}

func TestToBase62(t *testing.T) {
	tests := map[uint64]string{0: "0000000", 61: "000000Z", 62: "0000010", codeSpace - 1: "ZZZZZZZ"}
	for n, want := range tests {
		if got := toBase62(n); got != want {
			t.Errorf("toBase62(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestIsValidCode(t *testing.T) {
	tests := map[string]bool{
		"aB3x9Kq":  true,
		"0000000":  true,
		"abc":      false, // too short
		"aB3x9Kqq": false, // too long
		"aB3-9Kq":  false, // not Base62
		"api":      false,
	}
	for code, want := range tests {
		if got := isValidCode(code); got != want {
			t.Errorf("isValidCode(%q) = %v, want %v", code, got, want)
		}
	}
}
