package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// urlChecker reports whether a URL is on a list of known-bad sites.
// The handler depends on this interface so tests can use a fake.
type urlChecker interface {
	isUnsafe(ctx context.Context, target string) (bool, error)
}

var errNoAPIKey = errors.New("SAFE_BROWSING_API_KEY not set")

const (
	safeBrowsingEndpoint = "https://safebrowsing.googleapis.com/v4/threatMatches:find"
	safeBrowsingTimeout  = 3 * time.Second
)

// safeBrowsing checks URLs with the Google Safe Browsing Lookup API (v4).
type safeBrowsing struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func newSafeBrowsing(apiKey string) *safeBrowsing {
	return &safeBrowsing{
		apiKey:   apiKey,
		endpoint: safeBrowsingEndpoint,
		client:   &http.Client{Timeout: safeBrowsingTimeout},
	}
}

type threatEntry struct {
	URL string `json:"url"`
}

type safeBrowsingRequest struct {
	Client struct {
		ClientID      string `json:"clientId"`
		ClientVersion string `json:"clientVersion"`
	} `json:"client"`
	ThreatInfo struct {
		ThreatTypes      []string      `json:"threatTypes"`
		PlatformTypes    []string      `json:"platformTypes"`
		ThreatEntryTypes []string      `json:"threatEntryTypes"`
		ThreatEntries    []threatEntry `json:"threatEntries"`
	} `json:"threatInfo"`
}

// isUnsafe returns an error whenever it can't get a verdict (no key, timeout,
// bad status). The caller treats that as "don't create the link".
func (s *safeBrowsing) isUnsafe(ctx context.Context, target string) (bool, error) {
	if s.apiKey == "" {
		return false, errNoAPIKey
	}

	var body safeBrowsingRequest
	body.Client.ClientID = "url-shortener"
	body.Client.ClientVersion = "1.0"
	body.ThreatInfo.ThreatTypes = []string{"MALWARE", "SOCIAL_ENGINEERING", "UNWANTED_SOFTWARE", "POTENTIALLY_HARMFUL_APPLICATION"}
	body.ThreatInfo.PlatformTypes = []string{"ANY_PLATFORM"}
	body.ThreatInfo.ThreatEntryTypes = []string{"URL"}
	body.ThreatInfo.ThreatEntries = []threatEntry{{URL: target}}
	payload, err := json.Marshal(body)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.endpoint+"?key="+url.QueryEscape(s.apiKey), bytes.NewReader(payload))
	if err != nil {
		return false, errors.New("safe browsing: could not build request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		// The client's error text includes the request URL, which contains
		// the API key. Keep only the underlying cause.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return false, fmt.Errorf("safe browsing: %w", urlErr.Err)
		}
		return false, errors.New("safe browsing: request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("safe browsing: status %d", resp.StatusCode)
	}

	// No match is an empty object {}; a match has a non-empty "matches" list.
	var result struct {
		Matches []json.RawMessage `json:"matches"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return false, fmt.Errorf("safe browsing: bad response: %w", err)
	}
	return len(result.Matches) > 0, nil
}
