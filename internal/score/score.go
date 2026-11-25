// Package score turns structured company assessments and gated ads into scores.
package score

import (
	"math"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/config"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

// Company scores one assessment. Outreach needs the pass threshold and a local
// legal entity that is not known to be missing.
func Company(a model.Assessment, c *config.Config) model.Scored {
	sub := map[string]float64{
		"industry":    industry(a.IndustryTier, c),
		"maturity":    maturity(a.Headcount, a.Funding),
		"platform":    lookup(a.Platform, map[string]float64{"scale": 1, "modernise": 1, "both": 1, "product": 0.5, "maintenance": 0.2}, 0.4),
		"sponsorship": lookup(a.Sponsorship, map[string]float64{"offered": 1, "prior": 0.8, "none": 0}, 0.4),
	}
	s := model.Scored{A: a, Breakdown: map[string]float64{}, Flags: []string{}}
	for k, w := range c.CompanyWeights {
		part := w * sub[k]
		s.Breakdown[k] = part
		s.CompanyScore += part
	}
	s.CompanyScore = round1(s.CompanyScore)
	s.Outreach = s.CompanyScore >= c.PassThreshold
	if a.Sponsorship == "none" {
		s.Outreach = false
		s.Flags = append(s.Flags, "no_sponsorship")
	}
	switch a.LocalEntity {
	case "no":
		s.Outreach = false
		s.Flags = append(s.Flags, "no_local_entity")
	case "yes":
	default:
		s.Flags = append(s.Flags, "local_entity_unknown")
	}
	return s
}

// Role scores one gated ad on level and scope, salary and platform signals.
func Role(g model.GateResult, c *config.Config) float64 {
	x := g.X
	lvl := 0.0
	switch x.Level {
	case "senior", "lead":
		lvl = 0.8
	case "em":
		lvl = 0.65
	case "unknown":
		lvl = 0.6
	}
	if lvl > 0 {
		lvl += math.Min(0.05*float64(len(x.SeniorSignals)), 0.2)
		if c.PreferredYears > 0 && x.YearsRequired >= c.PreferredYears {
			lvl += 0.1
		}
		lvl = math.Min(lvl, 1)
	}
	sal := 0.4
	switch {
	case x.SalaryHiYear >= c.SalaryTarget:
		sal = 1
	case x.SalaryHiYear >= c.SalaryFloor:
		sal = 0.6
	case x.SalaryHiYear > 0:
		sal = 0
	}
	plat := 0.3
	switch n := len(x.PlatformSignals); {
	case n >= 2:
		plat = 1
	case n == 1:
		plat = 0.7
	}
	sub := map[string]float64{"level": lvl, "salary": sal, "platform": plat}
	total := 0.0
	for k, w := range c.RoleWeights {
		total += w * sub[k]
	}
	return round1(total)
}

func industry(tier string, c *config.Config) float64 {
	if v, ok := c.Industries[tier]; ok {
		return v
	}
	if v, ok := c.Industries["unknown"]; ok {
		return v
	}
	return 0.4
}

func maturity(headcount int, funding string) float64 {
	switch {
	case headcount >= 200, funding == "c_plus", funding == "public", funding == "group":
		return 1
	case headcount >= 50, funding == "b":
		return 0.6
	case headcount >= 1:
		return 0.2
	}
	return 0.4
}

func lookup(v string, m map[string]float64, unknown float64) float64 {
	if s, ok := m[v]; ok {
		return s
	}
	return unknown
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
