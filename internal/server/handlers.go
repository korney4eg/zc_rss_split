package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mmcdole/gofeed"
	ext "github.com/mmcdole/gofeed/extensions"

	"rsssplit/internal/cache"
	"rsssplit/internal/config"
)

// HandleIndex возвращает главную страницу со справкой
func HandleIndex(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "RSS splitter (config-driven). No per-request overrides.")
		fmt.Fprintf(w, "Source:  %s\nFormat:  %s\nRefresh: %s\n\n",
			cfg.Source, cfg.Format, cfg.Refresh)
		fmt.Fprintln(w, "Endpoints:")
		fmt.Fprintln(w, "  /feed?type=sdz|kabinet_lora|photo|zavtracast")
		fmt.Fprintln(w, "  /sdz   /kabinet_lora   /photo   /zavtracast")
	}
}

// HandleFeed обрабатывает запрос к /feed и его алиасам
func HandleFeed(cfg config.Config, cache *cache.SourceCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Определяем тип фида
		typ := r.URL.Query().Get("type")
		switch r.URL.Path {
		case "/sdz", "/kabinet_lora", "/photo", "/zavtracast":
			typ = strings.TrimPrefix(r.URL.Path, "/")
		}
		if typ == "" {
			typ = "zavtracast"
		}

		// Забираем данные из кэша/источника
		data, status, err := cache.Get(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("fetch %q: %v", cfg.Source, err), status)
			return
		}

		// Парсим входной фид
		fp := gofeed.NewParser()
		inFeed, err := fp.Parse(bytes.NewReader(data))
		if err != nil {
			http.Error(w, fmt.Sprintf("parse: %v", err), http.StatusBadRequest)
			return
		}

		// Фильтруем по типу
		out := filterFeedGF(inFeed, typ)
		applyTypeMetaGF(out, typ, cfg)

		// Рендерим
		switch strings.ToLower(cfg.Format) {
		case "rss":
			w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
			if err := renderRSS2WithExtensions(w, out); err != nil {
				http.Error(w, fmt.Sprintf("render rss: %v", err), http.StatusInternalServerError)
				return
			}
		case "atom":
			w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
			f := toGorilla(out)
			if xml, err := f.ToAtom(); err != nil {
				http.Error(w, fmt.Sprintf("render atom: %v", err), http.StatusInternalServerError)
			} else {
				_, _ = w.Write([]byte(xml))
			}
		default:
			http.Error(w, "server misconfig: unknown format", http.StatusInternalServerError)
		}
	}
}

// --- Фильтрация элементов ---

func filterFeedGF(in *gofeed.Feed, want string) *gofeed.Feed {
	cp := *in
	cp.Items = nil
	log.Printf("Looking for episodes for %s", want)
	for _, it := range in.Items {
		if selectType(it.Title) == want {
			log.Printf("Adding '%s' to %s", it.Title, want)
			cp.Items = append(cp.Items, it)
		}
	}
	return &cp
}

func applyTypeMetaGF(f *gofeed.Feed, typ string, cfg config.Config) {
	m, ok := cfg.TypeMetas[typ]
	if !ok {
		return
	}
	if strings.TrimSpace(m.Title) != "" {
		f.Title = m.Title
	}
	if strings.TrimSpace(m.Description) != "" {
		f.Description = m.Description
	}
	if strings.TrimSpace(m.ImageURL) != "" {
		if f.Image == nil {
			f.Image = &gofeed.Image{}
		}
		f.Image.URL = m.ImageURL
		if f.Extensions == nil {
			f.Extensions = ext.Extensions{}
		}
		if f.Extensions["itunes"] == nil {
			f.Extensions["itunes"] = map[string][]ext.Extension{}
		}
		f.Extensions["itunes"]["image"] = []ext.Extension{
			{
				Name:  "image",
				Attrs: map[string]string{"href": m.ImageURL},
			},
		}
	}
}

// --- Классификация выпусков ---

func selectType(title string) string {
	if strings.Contains(title, "СДЗ") {
		return "sdz"
	}
	if strings.HasPrefix(title, "Кабинет Лора ") ||
		title == "Завтракаст Special - Про Warhammer 40k" {
		return "kabinet_lora"
	}
	if strings.Contains(title, "Фотодушнила") {
		return "photo"
	}
	return "zavtracast"
}
