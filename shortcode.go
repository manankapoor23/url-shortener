package main

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
)

const (
	codeLength  = 7
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// codeSpace is 62^7 ≈ 3.5 trillion, the number of possible codes.
var codeSpace = func() uint64 {
	n := uint64(1)
	for i := 0; i < codeLength; i++ {
		n *= 62
	}
	return n
}()

// generateCode returns the short code for url on the given attempt.
// Attempt 0 hashes the URL alone. Retries hash url + attempt, so the retry
// sequence is deterministic and a resubmitted URL finds its existing code.
func generateCode(url string, attempt int) string {
	input := url
	if attempt > 0 {
		// A NUL byte can't appear in a parsed URL, so url+attempt can never
		// equal another URL's attempt-0 input.
		input = url + "\x00" + strconv.Itoa(attempt)
	}
	sum := sha256.Sum256([]byte(input))
	n := binary.BigEndian.Uint64(sum[:8]) % codeSpace
	return toBase62(n)
}

// toBase62 renders n as exactly codeLength Base62 characters, zero-padded.
func toBase62(n uint64) string {
	buf := make([]byte, codeLength)
	for i := codeLength - 1; i >= 0; i-- {
		buf[i] = base62Chars[n%62]
		n /= 62
	}
	return string(buf)
}

// isValidCode reports whether code has the shape of a code we could issue.
func isValidCode(code string) bool {
	if len(code) != codeLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
			return false
		}
	}
	return true
}
