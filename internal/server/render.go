package server

import (
	"encoding/xml"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
	ext "github.com/mmcdole/gofeed/extensions"
)

// --- известные namespace'ы ---
var rssNS = map[string]string{
	"itunes":     "http://www.itunes.com/dtds/podcast-1.0.dtd",
	"atom":       "http://www.w3.org/2005/Atom",
	"content":    "http://purl.org/rss/1.0/modules/content/",
	"googleplay": "http://www.google.com/schemas/play-podcasts/1.0",
	"media":      "http://search.yahoo.com/mrss/",
}

// Рендер RSS 2.0 с расширениями
func renderRSS2WithExtensions(w io.Writer, f *gofeed.Feed) error {
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")

	// Собираем префиксы
	prefixes := collectPrefixes(f)

	// <rss>
	rssStart := xml.StartElement{Name: xml.Name{Local: "rss"}, Attr: []xml.Attr{
		{Name: xml.Name{Local: "version"}, Value: "2.0"},
	}}
	for _, p := range prefixes {
		if uri, ok := rssNS[p]; ok {
			rssStart.Attr = append(rssStart.Attr, xml.Attr{
				Name:  xml.Name{Local: "xmlns:" + p},
				Value: uri,
			})
		}
	}
	if err := enc.EncodeToken(rssStart); err != nil {
		return err
	}

	// <channel>
	if err := enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "channel"}}); err != nil {
		return err
	}

	// основные поля канала
	encodeEl(enc, "title", f.Title)
	encodeEl(enc, "link", f.Link)
	encodeEl(enc, "description", coalesce(f.Description, f.Title))
	if f.Language != "" {
		encodeEl(enc, "language", f.Language)
	}
	if f.PublishedParsed != nil && !f.PublishedParsed.IsZero() {
		encodeEl(enc, "pubDate", f.PublishedParsed.Format(time.RFC1123Z))
	}
	if f.UpdatedParsed != nil && !f.UpdatedParsed.IsZero() {
		encodeEl(enc, "lastBuildDate", f.UpdatedParsed.Format(time.RFC1123Z))
	}
	if f.Image != nil && strings.TrimSpace(f.Image.URL) != "" {
		imgStart := xml.StartElement{Name: xml.Name{Local: "image"}}
		enc.EncodeToken(imgStart)
		encodeEl(enc, "url", f.Image.URL)
		encodeEl(enc, "title", coalesce(f.Image.Title, f.Title))
		encodeEl(enc, "link", coalesce(f.Image.URL, f.Link))
		enc.EncodeToken(xml.EndElement{Name: imgStart.Name})
	}

	// экстеншены канала
	writeExtensions(enc, f.Extensions)

	// элементы
	for _, it := range f.Items {
		if it == nil {
			continue
		}
		if err := enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "item"}}); err != nil {
			return err
		}
		encodeEl(enc, "title", it.Title)
		encodeEl(enc, "link", it.Link)
		encodeEl(enc, "description", coalesce(it.Description, it.Content))

		if it.PublishedParsed != nil && !it.PublishedParsed.IsZero() {
			encodeEl(enc, "pubDate", it.PublishedParsed.Format(time.RFC1123Z))
		} else if it.UpdatedParsed != nil && !it.UpdatedParsed.IsZero() {
			encodeEl(enc, "pubDate", it.UpdatedParsed.Format(time.RFC1123Z))
		}

		if strings.TrimSpace(it.GUID) != "" {
			start := xml.StartElement{Name: xml.Name{Local: "guid"}}
			if !strings.HasPrefix(strings.ToLower(it.GUID), "http://") &&
				!strings.HasPrefix(strings.ToLower(it.GUID), "https://") {
				start.Attr = append(start.Attr, xml.Attr{
					Name:  xml.Name{Local: "isPermaLink"},
					Value: "false",
				})
			}
			enc.EncodeToken(start)
			enc.EncodeToken(xml.CharData([]byte(it.GUID)))
			enc.EncodeToken(xml.EndElement{Name: start.Name})
		}

		if len(it.Enclosures) > 0 && it.Enclosures[0] != nil {
			enc0 := it.Enclosures[0]
			start := xml.StartElement{
				Name: xml.Name{Local: "enclosure"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "url"}, Value: enc0.URL},
				},
			}
			if enc0.Length != "" {
				start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "length"}, Value: enc0.Length})
			}
			if enc0.Type != "" {
				start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: enc0.Type})
			}
			enc.EncodeToken(start)
			enc.EncodeToken(xml.EndElement{Name: start.Name})
		}

		writeExtensions(enc, it.Extensions)
		enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "item"}})
	}

	// закрывающие теги
	enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "channel"}})
	enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "rss"}})
	return enc.Flush()
}

// encodeEl — безопасное добавление xml-элемента
func encodeEl(enc *xml.Encoder, name, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	start := xml.StartElement{Name: xml.Name{Local: name}}
	_ = enc.EncodeToken(start)
	_ = enc.EncodeToken(xml.CharData([]byte(value)))
	_ = enc.EncodeToken(xml.EndElement{Name: start.Name})
}

// writeExtensions пишет расширения в RSS
func writeExtensions(enc *xml.Encoder, exts ext.Extensions) {
	if len(exts) == 0 {
		return
	}
	prefixes := make([]string, 0, len(exts))
	for p := range exts {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	for _, p := range prefixes {
		ns := exts[p]
		if ns == nil {
			continue
		}
		names := make([]string, 0, len(ns))
		for n := range ns {
			names = append(names, n)
		}
		sort.Strings(names)

		for _, n := range names {
			for _, e := range ns[n] {
				writeExtension(enc, p, e)
			}
		}
	}
}

func writeExtension(enc *xml.Encoder, prefix string, e ext.Extension) {
	start := xml.StartElement{Name: xml.Name{Local: prefix + ":" + e.Name}}
	for k, v := range e.Attrs {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: k}, Value: v})
	}
	_ = enc.EncodeToken(start)

	if strings.TrimSpace(e.Value) != "" {
		_ = enc.EncodeToken(xml.CharData([]byte(e.Value)))
	}
	if len(e.Children) > 0 {
		childNames := make([]string, 0, len(e.Children))
		for cn := range e.Children {
			childNames = append(childNames, cn)
		}
		sort.Strings(childNames)

		for _, cn := range childNames {
			for _, ce := range e.Children[cn] {
				writeExtension(enc, prefix, ce)
			}
		}
	}
	_ = enc.EncodeToken(xml.EndElement{Name: start.Name})
}

// collectPrefixes собирает namespace-префиксы
func collectPrefixes(f *gofeed.Feed) []string {
	seen := map[string]bool{}
	add := func(m map[string]map[string][]ext.Extension) {
		for p := range m {
			seen[p] = true
		}
	}
	if f.Extensions != nil {
		add(f.Extensions)
	}
	for _, it := range f.Items {
		if it != nil && it.Extensions != nil {
			add(it.Extensions)
		}
	}
	var out []string
	for p := range seen {
		if _, ok := rssNS[p]; ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// toGorilla — конвертирует gofeed.Feed в gorilla/feeds.Feed (для Atom)
func toGorilla(in *gofeed.Feed) *feeds.Feed {
	f := &feeds.Feed{
		Title:       in.Title,
		Link:        &feeds.Link{Href: in.Link},
		Description: coalesce(in.Description, in.Title),
		Created:     firstNonZeroTime(in.PublishedParsed, in.UpdatedParsed),
		Updated:     firstNonZeroTime(in.UpdatedParsed, in.PublishedParsed),
	}
	if in.Author != nil {
		f.Author = &feeds.Author{Name: in.Author.Name, Email: in.Author.Email}
	}
	if in.Image != nil {
		f.Image = &feeds.Image{Url: in.Image.URL}
	}
	items := make([]*feeds.Item, 0, len(in.Items))
	for _, it := range in.Items {
		fi := &feeds.Item{
			Title:       it.Title,
			Link:        &feeds.Link{Href: it.Link},
			Description: coalesce(it.Description, it.Content),
			Content:     it.Content,
			Created:     firstNonZeroTime(it.PublishedParsed, it.UpdatedParsed, in.PublishedParsed),
			Updated:     firstNonZeroTime(it.UpdatedParsed, it.PublishedParsed),
			Id:          coalesce(it.GUID, it.Link),
		}
		if it.Author != nil {
			fi.Author = &feeds.Author{Name: it.Author.Name, Email: it.Author.Email}
		}
		if len(it.Enclosures) > 0 && it.Enclosures[0] != nil {
			enc := it.Enclosures[0]
			fi.Enclosure = &feeds.Enclosure{Url: enc.URL, Length: enc.Length, Type: enc.Type}
		}
		items = append(items, fi)
	}
	f.Items = items
	return f
}
