package server

import (
	"strings"
	"time"
)

// coalesce возвращает первый непустой (после TrimSpace) аргумент или ""
func coalesce(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// firstNonZeroTime возвращает первое ненулевое время из переданных указателей,
// либо zero-time если ничего нет.
func firstNonZeroTime(ts ...*time.Time) time.Time {
	for _, t := range ts {
		if t != nil && !t.IsZero() {
			return *t
		}
	}
	return time.Time{}
}
