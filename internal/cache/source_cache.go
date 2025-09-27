package cache

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"rsssplit/internal/config"
)

type SourceCache struct {
	cfg     config.Config
	mu      sync.RWMutex
	data    []byte
	etag    string
	lastMod string
	started bool
}

func NewSourceCache(cfg config.Config) *SourceCache { return &SourceCache{cfg: cfg} }

func (c *SourceCache) Start(ctx context.Context) {
	if c.cfg.Refresh <= 0 || c.started {
		return
	}
	c.started = true

	if _, _, err := c.fetch(ctx); err != nil {
		log.Printf("[ERROR] initial fetch error: %v", err)
	}

	t := time.NewTicker(c.cfg.Refresh)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, _, err := c.fetch(context.Background()); err != nil {
					log.Printf("[ERROR] background fetch error: %v", err)
				}
			}
		}
	}()
}

func (c *SourceCache) Get(ctx context.Context) ([]byte, int, error) {
	if c.cfg.Refresh <= 0 {
		return c.fetch(ctx)
	}
	c.mu.RLock()
	if len(c.data) > 0 {
		defer c.mu.RUnlock()
		return append([]byte(nil), c.data...), http.StatusOK, nil
	}
	c.mu.RUnlock()
	return c.fetch(ctx)
}

func (c *SourceCache) fetch(ctx context.Context) ([]byte, int, error) {
	src := c.cfg.Source
	log.Printf("Getting data from %s", src)
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		req, err := http.NewRequestWithContext(ctx, "GET", src, nil)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		if c.cfg.UserAgent != "" {
			req.Header.Set("User-Agent", c.cfg.UserAgent)
		}
		c.mu.RLock()
		if c.etag != "" {
			req.Header.Set("If-None-Match", c.etag)
		}
		if c.lastMod != "" {
			req.Header.Set("If-Modified-Since", c.lastMod)
		}
		c.mu.RUnlock()

		client := &http.Client{Timeout: c.cfg.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			return nil, http.StatusBadGateway, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotModified {
			c.mu.RLock()
			defer c.mu.RUnlock()
			if len(c.data) == 0 {
				return nil, http.StatusBadGateway, fmt.Errorf("304 but no cached body yet")
			}
			return append([]byte(nil), c.data...), http.StatusOK, nil
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, resp.StatusCode, fmt.Errorf("upstream HTTP %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, http.StatusBadGateway, err
		}
		log.Printf("Successfully read http feed")

		c.mu.Lock()
		c.data = append([]byte(nil), body...)
		c.etag = resp.Header.Get("ETag")
		c.lastMod = resp.Header.Get("Last-Modified")
		c.mu.Unlock()

		return append([]byte(nil), body...), http.StatusOK, nil
	}

	path := strings.TrimPrefix(src, "file://")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	log.Printf("Successfully read local feed")
	c.mu.Lock()
	c.data = append([]byte(nil), b...)
	c.etag = ""
	c.lastMod = ""
	c.mu.Unlock()
	return append([]byte(nil), b...), http.StatusOK, nil
}
