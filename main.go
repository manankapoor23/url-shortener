package main

import (
	_ "embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

//go:embed static/index.html
var indexHTML []byte

// loadConfig reads settings from .env (if present) and the environment.
// Real environment variables win over .env, so a host's settings take priority.
func loadConfig() *viper.Viper {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalf("reading .env: %v", err)
	}
	v.AutomaticEnv()
	v.SetDefault("PORT", "8080")
	return v
}

func main() {
	cfg := loadConfig()
	port := cfg.GetString("PORT")

	// BASE_URL is the public address short links are built from.
	baseURL := strings.TrimRight(cfg.GetString("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		log.Fatalf("invalid BASE_URL %q", baseURL)
	}

	apiKey := cfg.GetString("SAFE_BROWSING_API_KEY")
	if apiKey == "" {
		// Start anyway so existing links keep redirecting.
		log.Print("SAFE_BROWSING_API_KEY not set: link creation will return 503")
	}

	srv := &server{
		store:   newStore(),
		checker: newSafeBrowsing(apiKey),
		baseURL: baseURL,
		ownHost: base.Hostname(),
	}

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
