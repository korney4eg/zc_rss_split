package server

import (
	"log"
	"net/http"
	"time"

	"rsssplit/internal/cache"
	"rsssplit/internal/config"
)

func New(cfg config.Config, cache *cache.SourceCache) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HandleIndex(cfg))
	mux.HandleFunc("/feed", HandleFeed(cfg, cache))
	mux.HandleFunc("/sdz", HandleFeed(cfg, cache))
	mux.HandleFunc("/kabinet_lora", HandleFeed(cfg, cache))
	mux.HandleFunc("/photo", HandleFeed(cfg, cache))
	mux.HandleFunc("/zavtracast", HandleFeed(cfg, cache))

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.UserAgent(), r.Method, r.URL.String(), time.Since(start))
	})
}
