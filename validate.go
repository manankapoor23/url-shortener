package main

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const maxURLLength = 2048

var (
	errNotURL      = errors.New("must be a full URL like https://example.com")
	errScheme      = errors.New("only http and https URLs are allowed")
	errTooLong     = errors.New("URL is too long (max 2048 characters)")
	errCredentials = errors.New("URLs containing a username or password are not allowed")
	errOwnDomain   = errors.New("URLs pointing to this shortener are not allowed")
	errPrivateHost = errors.New("URLs pointing to localhost or private networks are not allowed")
)

// validateURL applies the layer 1 input rules and returns the normalized URL.
// ownHost is this shortener's hostname (rule 5).
func validateURL(raw, ownHost string) (string, error) {
	// Rule 7: normalize before anything else, so checks see the canonical form.
	// A string with spaces, tabs, and newlines at both ends is raw
	raw = strings.TrimSpace(raw) // we are going to remove the whitespace and all from it /n , /t, /r etc

	// Rule 3: check the maxlength of the input url
	if len(raw) > maxURLLength {
		return "", errTooLong
	}

	// Rule 1: must parse as a full URL. url.Parse lowercases the scheme.
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return "", errNotURL
	}

	// Rule 2: http/https only. Blocks javascript:, data:, file:, ...
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errScheme
	}

	// Rule 1 (continued): a web URL needs a host.
	// u.Hostname() gives out the host without the port
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", errNotURL
	}
	// and u.Host gives out the full with the port
	u.Host = strings.ToLower(u.Host) // Rule 7: lowercase host.

	// Rule 4: no user:password@ part (e.g. https://paypal.com@evil.com).
	if u.User != nil {
		return "", errCredentials
	}

	// Rule 5: not our own domain (no wrapping short links or loops).
	if ownHost != "" && host == strings.ToLower(ownHost) {
		return "", errOwnDomain
	}

	// Rule 6: not localhost or a private IP.
	if isPrivateHost(host) {
		return "", errPrivateHost
	}

	return u.String(), nil // returns a string and an error
}

// isPrivateHost is a function which reports whether host is localhost or an IP literal for a
// loopback, private or link-local address. It does not resolve DNS, so a
// domain that points to a private IP still passes.
func isPrivateHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
	}
	// Browsers read hosts like 2130706433, 0x7f000001 or 127.1 as IPv4
	// (127.0.0.1), but net.ParseIP doesn't. A host whose last label is a
	// number is never a real domain, so treat it as unsafe.
	labels := strings.Split(strings.TrimSuffix(host, "."), ".")
	return isNumericLabel(labels[len(labels)-1])
}

func isNumericLabel(label string) bool {
	if strings.HasPrefix(label, "0x") {
		label = label[2:]
		if label == "" {
			return true
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
				return false
			}
		}
		return true
	}
	if label == "" {
		return false
	}
	for i := 0; i < len(label); i++ {
		if label[i] < '0' || label[i] > '9' {
			return false
		}
	}
	return true
}
