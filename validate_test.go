package main

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateURLAccepts(t *testing.T) {
	tests := map[string]string{
		"https://example.com":              "https://example.com",
		"http://example.com/path?q=1#frag": "http://example.com/path?q=1#frag",
		"  HTTPS://Example.COM/CaseKept  ": "https://example.com/CaseKept", // rule 7
		"https://8.8.8.8/":                 "https://8.8.8.8/",             // public IP is fine
		"https://example.com:8443/a":       "https://example.com:8443/a",
	}
	for in, want := range tests {
		got, err := validateURL(in, "short.example")
		if err != nil || got != want {
			t.Errorf("validateURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestValidateURLRejects(t *testing.T) {
	tests := []struct {
		in   string
		want error
	}{
		{"", errNotURL},
		{"hello", errNotURL},
		{"www.example.com", errNotURL},
		{"https://", errNotURL},
		{"javascript:alert(document.cookie)", errScheme},
		{"data:text/html,<script>alert(1)</script>", errScheme},
		{"file:///etc/passwd", errScheme},
		{"ftp://example.com", errScheme},
		{"https://example.com/" + strings.Repeat("a", maxURLLength), errTooLong},
		{"https://user:pass@example.com", errCredentials},
		{"https://paypal.com@evil.com", errCredentials},
		{"https://short.example/aB3x9Kq", errOwnDomain},
		{"https://SHORT.example/aB3x9Kq", errOwnDomain},
		{"http://localhost:3000", errPrivateHost},
		{"http://api.localhost", errPrivateHost},
		{"http://127.0.0.1", errPrivateHost},
		{"http://192.168.1.1/admin", errPrivateHost},
		{"http://10.0.0.5", errPrivateHost},
		{"http://169.254.169.254/latest/meta-data", errPrivateHost},
		{"http://[::1]/", errPrivateHost},
		{"http://0.0.0.0", errPrivateHost},
		{"http://2130706433/", errPrivateHost}, // decimal form of 127.0.0.1
		{"http://0x7f000001/", errPrivateHost}, // hex form of 127.0.0.1
		{"http://127.1/", errPrivateHost},      // short form of 127.0.0.1
	}
	for _, tt := range tests {
		if _, err := validateURL(tt.in, "short.example"); !errors.Is(err, tt.want) {
			t.Errorf("validateURL(%q) error = %v, want %v", tt.in, err, tt.want)
		}
	}
}
