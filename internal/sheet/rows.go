// Package sheet turns pipeline results into rows for the Google Sheet tracker.
// Column order must match docs/sheet-schema.md.
package sheet

import (
	"math"
	"strings"
	"time"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

// CompanyColumns is the Companies tab header. status and notes are edited by hand.
var CompanyColumns = []string{
	"market", "company", "domain", "website", "industry_tier", "headcount", "funding", "platform",
	"sponsorship", "local_entity", "company_score", "outreach", "flags", "reason", "sources",
	"open_roles", "last_assessed", "status", "notes",
}

// LeadColumns is the Leads tab header. status and notes are edited by hand.
var LeadColumns = []string{
	"market", "posted", "company", "title", "city", "mode", "salary_lo", "salary_hi", "main_stack",
	"level", "role_score", "flags", "url", "source", "status", "notes",
}

// CompanyRows builds one Companies row per scored assessment.
func CompanyRows(scored []model.Scored, assessedOn time.Time) [][]any {
	rows := make([][]any, 0, len(scored))
	for _, s := range scored {
		a := s.A
		rows = append(rows, []any{
			a.Market, a.Company, a.Domain, a.Website, a.IndustryTier, a.Headcount, a.Funding, a.Platform,
			a.Sponsorship, a.LocalEntity, round1(s.CompanyScore), yesNo(s.Outreach), strings.Join(s.Flags, ", "),
			a.Reason, a.Sources, strings.Join(a.OpenRoleURLs, "\n"), assessedOn.Format("2006-01-02"), "new", "",
		})
	}
	return rows
}

// LeadRows builds one Leads row per gated ad, with its role score.
func LeadRows(gated []model.GateResult, roleScores []float64) [][]any {
	rows := make([][]any, 0, len(gated))
	for i, g := range gated {
		j, x := g.Job, g.X
		score := 0.0
		if i < len(roleScores) {
			score = round1(roleScores[i])
		}
		posted := ""
		if !j.Posted.IsZero() {
			posted = j.Posted.Format("2006-01-02")
		}
		rows = append(rows, []any{
			j.Market, posted, j.Company, j.Title, x.City, x.Mode, x.SalaryLoYear, x.SalaryHiYear, x.MainStack,
			x.Level, score, strings.Join(g.Flags, ", "), j.URL, j.Source, "new", "",
		})
	}
	return rows
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
