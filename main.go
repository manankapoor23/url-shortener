package main

import (
	_ "embed"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed static/index.html
var indexHTML []byte

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// BASE_URL is the public address short links are built from.
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		log.Fatalf("invalid BASE_URL %q", baseURL)
	}

	srv := &server{store: newStore(), baseURL: baseURL, ownHost: base.Hostname()}

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: srv.routes(),
		// Timeouts stop slow clients from holding connections open forever.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on :%s (short links use %s)", port, baseURL)
	log.Fatal(httpServer.ListenAndServe())
}
