// Package source converts raw job data from each board into model.Job.
package source

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

type normalizer func(item map[string]any) model.Job

var kinds = map[string]normalizer{
	"linkedin":     linkedin,
	"tecnoempleo":  tecnoempleo,
	"indeed_apify": indeedApify,
}

// Normalize parses raw (a JSON array of items, or {"items": [...]}) for one source kind.
func Normalize(kind string, raw []byte, market string) ([]model.Job, error) {
	if kind == "generic" {
		return generic(raw, market)
	}
	fn, ok := kinds[kind]
	if !ok {
		return nil, fmt.Errorf("unknown source kind: %s", kind)
	}
	var items []map[string]any
	if err := unmarshalItems(raw, &items); err != nil {
		return nil, fmt.Errorf("parse %s items: %w", kind, err)
	}
	jobs := make([]model.Job, 0, len(items))
	for _, it := range items {
		j := fn(it)
		j.Source, j.Market = kind, market
		if j.ID == "" {
			j.ID = j.URL
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func generic(raw []byte, market string) ([]model.Job, error) {
	var jobs []model.Job
	if err := unmarshalItems(raw, &jobs); err != nil {
		return nil, fmt.Errorf("parse generic jobs: %w", err)
	}
	for i := range jobs {
		if jobs[i].Market == "" {
			jobs[i].Market = market
		}
		if jobs[i].Source == "" {
			jobs[i].Source = "generic"
		}
		if jobs[i].ID == "" {
			jobs[i].ID = jobs[i].URL
		}
		jobs[i].Desc = clean(jobs[i].Desc)
	}
	return jobs, nil
}

// unmarshalItems accepts a bare array or an object with an "items" array.
func unmarshalItems(raw []byte, out any) error {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "{") {
		var wrap struct {
			Items json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(raw, &wrap); err != nil {
			return err
		}
		raw = wrap.Items
	}
	return json.Unmarshal(raw, out)
}

var tagRe = regexp.MustCompile(`<[^>]+>`)
var spaceRe = regexp.MustCompile(`[ \t]+`)

func clean(s string) string {
	s = html.UnescapeString(s)
	s = tagRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func str(it map[string]any, key string) string {
	switch v := it[key].(type) {
	case string:
		return v
	case float64:
		return strings.TrimSuffix(fmt.Sprintf("%f", v), ".000000")
	}
	return ""
}

func num(it map[string]any, key string) float64 {
	if v, ok := it[key].(float64); ok {
		return v
	}
	return 0
}

func first(it map[string]any, key string) string {
	if arr, ok := it[key].([]any); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok {
			return s
		}
	}
	return ""
}

func date(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	if len(s) >= 10 {
		if t, err := time.Parse("2006-01-02", s[:10]); err == nil {
			return t
		}
	}
	return time.Time{}
}

func period(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "year"), strings.Contains(s, "año"), strings.Contains(s, "annual"):
		return "year"
	case strings.Contains(s, "month"), strings.Contains(s, "mes"):
		return "month"
	case strings.Contains(s, "hour"), strings.Contains(s, "hora"):
		return "hour"
	case s == "":
		return ""
	}
	return "unknown"
}
