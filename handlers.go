package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// maxRequestBytes caps the create request body before it is parsed.
const maxRequestBytes = 4 << 10 // 4 KB

type server struct {
	store   *store
	baseURL string // e.g. http://localhost:8080, no trailing slash
	ownHost string // hostname of baseURL, for validation rule 5
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /api/shorten", s.handleShorten)
	mux.HandleFunc("GET /{code}", s.handleRedirect)
	return mux
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

func (s *server) handleShorten(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)

	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request is too large")
			return
		}
		writeError(w, http.StatusBadRequest, `request body must be JSON like {"url": "https://example.com"}`)
		return
	}

	normalized, err := validateURL(req.URL, s.ownHost)
	if err != nil {
		// Validation errors describe the user's own input, so they're safe to show.
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	code, err := s.store.save(normalized)
	if err != nil {
		log.Printf("save %q: %v", normalized, err)
		writeError(w, http.StatusInternalServerError, "could not create a short link, please try again")
		return
	}

	writeJSON(w, http.StatusOK, shortenResponse{Code: code, ShortURL: s.baseURL + "/" + code})
}

// handleRedirect sends known codes to their URL with a 302. Malformed,
// unknown and vanished codes all get the same generic 404, so responses
// don't reveal which codes ever existed.
func (s *server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if isValidCode(code) {
		if target, ok := s.store.lookup(code); ok {
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
	}
	http.NotFound(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
