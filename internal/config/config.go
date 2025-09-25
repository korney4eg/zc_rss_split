package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Source    string     `yaml:"source"`
	Format    string     `yaml:"format"`
	Addr      string     `yaml:"addr"`
	Timeout   string     `yaml:"timeout"`
	UserAgent string     `yaml:"user_agent"`
	Refresh   string     `yaml:"refresh"`
	Types     TypesBlock `yaml:"types"`
}

type TypesBlock struct {
	SDZ         TypeMeta `yaml:"sdz"`
	KabinetLora TypeMeta `yaml:"kabinet_lora"`
	Photo       TypeMeta `yaml:"photo"`
	Zavtracast  TypeMeta `yaml:"zavtracast"`
}

type TypeMeta struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	ImageURL    string `yaml:"image"`
}

type Config struct {
	Source    string
	Format    string
	Addr      string
	Timeout   time.Duration
	UserAgent string
	Refresh   time.Duration
	TypeMetas map[string]TypeMeta
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var fc fileConfig
	if err := yaml.Unmarshal(raw, &fc); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}

	if strings.TrimSpace(fc.Source) == "" {
		return Config{}, fmt.Errorf("config: source is required")
	}
	format := strings.ToLower(strings.TrimSpace(fc.Format))
	if format == "" {
		format = "rss"
	}
	if format != "rss" && format != "atom" {
		return Config{}, fmt.Errorf("config: invalid format %q", fc.Format)
	}
	addr := fc.Addr
	if addr == "" {
		addr = ":8080"
	}
	ua := fc.UserAgent
	if strings.TrimSpace(ua) == "" {
		ua = "rsssplit/1.5 (+https://example.com)"
	}
	timeout := parseDurDefault(fc.Timeout, 20*time.Second)
	refresh := parseDurDefault(fc.Refresh, 0)

	return Config{
		Source:    fc.Source,
		Format:    format,
		Addr:      addr,
		Timeout:   timeout,
		UserAgent: ua,
		Refresh:   refresh,
		TypeMetas: map[string]TypeMeta{
			"sdz":          fc.Types.SDZ,
			"kabinet_lora": fc.Types.KabinetLora,
			"photo":        fc.Types.Photo,
			"zavtracast":   fc.Types.Zavtracast,
		},
	}, nil
}

func parseDurDefault(s string, def time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("invalid duration %q, using default %s", s, def)
		return def
	}
	return d
}
