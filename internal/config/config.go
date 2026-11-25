// Package config loads the global settings and one market profile.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// SourceCfg describes one place the runner pulls ads from.
type SourceCfg struct {
	Name    string         `yaml:"name"`
	Kind    string         `yaml:"kind"`    // linkedin|tecnoempleo|indeed_apify|generic
	Via     string         `yaml:"via"`     // apify|indeed_connector
	Actor   string         `yaml:"actor"`   // Apify actor id, when Via == apify
	Tier    string         `yaml:"tier"`    // free|paid: minimum tier that uses this source
	Queries []string       `yaml:"queries"` // search terms
	Input   map[string]any `yaml:"input"`   // actor input template
}

// Config is global settings merged with one market profile.
type Config struct {
	Market       string `yaml:"market"`
	Country      string `yaml:"country"`
	Currency     string `yaml:"currency"`
	OutreachLang string `yaml:"outreach_lang"`
	Tier         string `yaml:"tier"` // free|paid

	SalaryFloor  float64 `yaml:"salary_floor"`
	SalaryTarget float64 `yaml:"salary_target"`

	Cities          map[string][]string `yaml:"cities"`        // canonical city -> allowed modes; "*" = other cities
	MetroAliases    map[string][]string `yaml:"metro_aliases"` // canonical city -> substrings that map to it
	RemoteWords     []string            `yaml:"remote_words"`
	HybridWords     []string            `yaml:"hybrid_words"`
	OnsiteWords     []string            `yaml:"onsite_words"`
	RemoteDescWords []string            `yaml:"remote_desc_words"` // remote wording strong enough to trust in a description

	RightsRequired []string `yaml:"rights_required"`
	BasedIn        []string `yaml:"based_in"`
	SponsorOffered []string `yaml:"sponsor_offered"`
	SponsorRefused []string `yaml:"sponsor_refused"`

	JuniorTitle    []string `yaml:"junior_title"`
	SeniorTitle    []string `yaml:"senior_title"`
	LeadTitle      []string `yaml:"lead_title"`
	EMTitle        []string `yaml:"em_title"`
	SeniorYears    int      `yaml:"senior_years"`    // minimum years: implies senior; a lower stated minimum is excluded
	PreferredYears int      `yaml:"preferred_years"` // years that rank a role higher

	Stacks          map[string]string `yaml:"stacks"` // stack key -> regex
	TargetStacks    []string          `yaml:"target_stacks"`
	StackNoise      []string          `yaml:"stack_noise"` // phrases removed before stack matching ("go to", "go live")
	PlatformSignals map[string]string `yaml:"platform_signals"`
	SeniorSignals   map[string]string `yaml:"senior_signals"`

	Industries     map[string]float64 `yaml:"industries"`
	CompanyWeights map[string]float64 `yaml:"company_weights"`
	RoleWeights    map[string]float64 `yaml:"role_weights"`
	PassThreshold  float64            `yaml:"pass_threshold"`
	ReassessDays   int                `yaml:"reassess_days"`
	DailyTarget    int                `yaml:"daily_target"`
	MaxContacts    int                `yaml:"max_contacts"`

	Sources          []SourceCfg `yaml:"sources"`
	DiscoveryQueries []string    `yaml:"discovery_queries"`
}

// Load reads dir/global.yaml, then overlays dir/markets/<market>.yaml.
// Values set in the market file win. Maps are merged key by key; lists are replaced.
func Load(dir, market string) (*Config, error) {
	c := &Config{}
	g, err := os.ReadFile(filepath.Join(dir, "global.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read global config: %w", err)
	}
	if err := yaml.Unmarshal(g, c); err != nil {
		return nil, fmt.Errorf("parse global config: %w", err)
	}
	m, err := os.ReadFile(filepath.Join(dir, "markets", market+".yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("market profile not found: %s", market)
		}
		return nil, fmt.Errorf("read market profile: %w", err)
	}
	global := *c
	if err := yaml.Unmarshal(m, c); err != nil {
		return nil, fmt.Errorf("parse market profile %s: %w", market, err)
	}
	// A market map replaces the global one on unmarshal; put back the global
	// keys the market did not set, so overriding one weight keeps the others.
	c.Cities = mergeMap(global.Cities, c.Cities)
	c.MetroAliases = mergeMap(global.MetroAliases, c.MetroAliases)
	c.Stacks = mergeMap(global.Stacks, c.Stacks)
	c.PlatformSignals = mergeMap(global.PlatformSignals, c.PlatformSignals)
	c.SeniorSignals = mergeMap(global.SeniorSignals, c.SeniorSignals)
	c.Industries = mergeMap(global.Industries, c.Industries)
	c.CompanyWeights = mergeMap(global.CompanyWeights, c.CompanyWeights)
	c.RoleWeights = mergeMap(global.RoleWeights, c.RoleWeights)
	if c.Market == "" {
		c.Market = market
	}
	return c, nil
}

func mergeMap[V any](global, market map[string]V) map[string]V {
	if market == nil {
		return global
	}
	for k, v := range global {
		if _, ok := market[k]; !ok {
			market[k] = v
		}
	}
	return market
}
