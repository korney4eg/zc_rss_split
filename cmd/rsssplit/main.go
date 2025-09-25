package main

import (
	"context"
	"flag"
	"log"

	"rsssplit/internal/cache"
	"rsssplit/internal/config"
	"rsssplit/internal/server"
)

func main() {
	confPath := flag.String("config", "config.yaml", "Path to YAML config")
	flag.Parse()

	cfg, err := config.Load(*confPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := cache.NewSourceCache(cfg)
	cache.Start(ctx)

	srv := server.New(cfg, cache)
	log.Printf("listening on %s (src=%s, format=%s, refresh=%s)",
		cfg.Addr, cfg.Source, cfg.Format, cfg.Refresh)
	log.Fatal(srv.ListenAndServe())
}
