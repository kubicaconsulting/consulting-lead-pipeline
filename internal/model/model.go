// Package model holds the data shapes shared by every pipeline step.
package model

import "time"

// Job is one job ad, normalised from any source.
type Job struct {
	ID            string    `json:"id"`
	Source        string    `json:"source"`
	Market        string    `json:"market"`
	Title         string    `json:"title"`
	Company       string    `json:"company"`
	CompanyDomain string    `json:"company_domain,omitempty"`
	City          string    `json:"city,omitempty"`
	Location      string    `json:"location,omitempty"`
	URL           string    `json:"url"`
	Lang          string    `json:"lang,omitempty"`
	Desc          string    `json:"desc"`
	Posted        time.Time `json:"posted"`
	SalaryText    string    `json:"salary_text,omitempty"`
	SalaryLo      float64   `json:"salary_lo,omitempty"`
	SalaryHi      float64   `json:"salary_hi,omitempty"`
	SalaryPeriod  string    `json:"salary_period,omitempty"` // year|month|unknown
	Headcount     string    `json:"headcount,omitempty"`
	Industry      string    `json:"industry,omitempty"`
	WorkMode      string    `json:"work_mode,omitempty"` // as given by the board, if any
}

// Extracted holds fields derived from a Job's text.
type Extracted struct {
	Stacks          []string `json:"stacks"`
	MainStack       string   `json:"main_stack"`
	Level           string   `json:"level"`          // junior|mid|senior|lead|em|unknown
	YearsRequired   int      `json:"years_required"` // lowest stated minimum years of experience, 0 if none
	Mode            string   `json:"mode"`           // onsite|hybrid|remote|unknown
	City            string   `json:"city"`           // canonical city key, or "" when unknown
	Rights          string   `json:"rights"`         // required|based_in|none
	Sponsorship     string   `json:"sponsorship"`    // offered|refused|unknown
	SalaryLoYear    float64  `json:"salary_lo_year"`
	SalaryHiYear    float64  `json:"salary_hi_year"`
	PlatformSignals []string `json:"platform_signals"`
	SeniorSignals   []string `json:"senior_signals"`
	Flags           []string `json:"flags"`
}

// GateResult is a Job after extraction and hard gates.
type GateResult struct {
	Job       Job       `json:"job"`
	X         Extracted `json:"extracted"`
	Pass      bool      `json:"pass"`
	Reasons   []string  `json:"reasons"`
	Flags     []string  `json:"flags"`
	DedupeKey string    `json:"dedupe_key"`
}

// Assessment is Claude's structured research on one company.
type Assessment struct {
	Company      string   `json:"company"`
	Domain       string   `json:"domain"`
	Market       string   `json:"market"`
	Website      string   `json:"website"`
	IndustryTier string   `json:"industry_tier"` // key in Config.Industries, or "unknown"
	Headcount    int      `json:"headcount"`     // 0 = unknown
	Funding      string   `json:"funding"`       // seed|a|b|c_plus|public|group|bootstrapped|unknown
	Platform     string   `json:"platform"`      // scale|modernise|both|product|maintenance|unknown
	Sponsorship  string   `json:"sponsorship"`   // offered|prior|unknown|none
	LocalEntity  string   `json:"local_entity"`  // yes|no|unknown
	OpenRoleURLs []string `json:"open_role_urls"`
	Reason       string   `json:"reason"`
	Sources      string   `json:"sources"`
}

// Scored is an Assessment with its company score.
type Scored struct {
	A            Assessment         `json:"assessment"`
	CompanyScore float64            `json:"company_score"`
	Breakdown    map[string]float64 `json:"breakdown"`
	Outreach     bool               `json:"outreach"`
	Flags        []string           `json:"flags"`
}

// SeenCompany is a company already in the tracker.
type SeenCompany struct {
	Key          string    `json:"key"`  // domain if known, else normalised name
	Name         string    `json:"name"` // company name as written in the tracker
	LastAssessed time.Time `json:"last_assessed"`
	RoleURLs     []string  `json:"role_urls"`
}
