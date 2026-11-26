// Package gate applies the market's hard filters and removes duplicate companies.
package gate

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/config"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/extract"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

var (
	legalRe  = regexp.MustCompile(`(?i)\b(s\.?l\.?u?|s\.?a\.?u?|s\.?l\.?l|inc|ltd|llc|gmbh|bv|b\.v\.|plc|corp|group|grupo)\b\.?`)
	nonAlpha = regexp.MustCompile(`[^a-z]`)
	schemeRe = regexp.MustCompile(`^(https?://)?(www\.)?`)
)

// Key is the dedupe key: the bare domain when known, else the company name
// lowercased without legal suffixes, digits, spaces or punctuation.
func Key(company, domain string) string {
	if d := strings.TrimSpace(domain); d != "" {
		d = schemeRe.ReplaceAllString(strings.ToLower(d), "")
		if i := strings.IndexAny(d, "/?#"); i >= 0 {
			d = d[:i]
		}
		return d
	}
	n := legalRe.ReplaceAllString(company, " ")
	return nonAlpha.ReplaceAllString(strings.ToLower(n), "")
}

// Same reports whether two keys name the same company: equal, or one name key
// starts with the other and the shorter is at least 4 letters ("glovo" / "glovoapp").
func Same(a, b string) bool {
	if a == b {
		return a != ""
	}
	if strings.Contains(a, ".") || strings.Contains(b, ".") {
		return false
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	return len(a) >= 4 && strings.HasPrefix(b, a)
}

// Run extracts fields and applies, in order: work rights, seniority, stack,
// location/mode, salary, recently assessed. The first failure is the reason;
// flags are collected either way. Duplicate company+title pairs in the batch
// keep only the job with the longest description.
func Run(jobs []model.Job, seen []model.SeenCompany, now time.Time, c *config.Config) (passed, excluded []model.GateResult) {
	for _, j := range dedupeBatch(jobs) {
		g := model.GateResult{Job: j, X: extract.Run(j, c), DedupeKey: Key(j.Company, j.CompanyDomain), Reasons: []string{}}
		g.Flags = append([]string{}, g.X.Flags...)
		fail := func(r string) {
			if len(g.Reasons) == 0 {
				g.Reasons = append(g.Reasons, r)
			}
		}
		x := g.X

		if x.Rights == "required" {
			fail("requires_work_rights")
		} else if x.Rights == "based_in" {
			g.Flags = append(g.Flags, "based_in_wording")
		}
		if x.Sponsorship == "refused" {
			fail("sponsorship_refused")
		}

		switch x.Level {
		case "junior", "mid":
			fail("seniority")
		case "unknown":
			g.Flags = append(g.Flags, "level_unknown")
		}

		switch {
		case x.YearsRequired > 0 && x.YearsRequired < c.SeniorYears:
			fail("experience_below_min")
		case c.PreferredYears > 0 && x.YearsRequired >= c.PreferredYears:
			g.Flags = append(g.Flags, "experience_10_plus")
		}

		if !stackOK(x, c) {
			fail("stack")
		}

		allowed, ok := c.Cities[x.City]
		if !ok || x.City == "" {
			allowed = c.Cities["*"]
		}
		if x.Mode == "unknown" {
			if !slices.Contains(allowed, "onsite") {
				g.Flags = append(g.Flags, "mode_unknown")
			}
		} else if !slices.Contains(allowed, x.Mode) {
			fail("location_mode")
		}

		switch {
		case x.SalaryHiYear == 0 && x.SalaryLoYear == 0:
			g.Flags = append(g.Flags, "salary_unknown")
		case x.SalaryHiYear == 0:
			g.Flags = append(g.Flags, "salary_min_only") // "from X": the top of the range is unknown
		case x.SalaryHiYear < c.SalaryFloor:
			fail("salary_below_floor")
		case x.SalaryHiYear < c.SalaryTarget:
			g.Flags = append(g.Flags, "salary_below_target")
		}

		if recentlyAssessed(j, seen, now, c.ReassessDays) {
			fail("recently_assessed")
		}

		g.Pass = len(g.Reasons) == 0
		if g.Pass {
			passed = append(passed, g)
		} else {
			excluded = append(excluded, g)
		}
	}
	return passed, excluded
}

func stackOK(x model.Extracted, c *config.Config) bool {
	hasTarget := false
	for _, s := range x.Stacks {
		if slices.Contains(c.TargetStacks, s) {
			hasTarget = true
		}
	}
	if !hasTarget {
		return false
	}
	return x.MainStack == "mixed" || x.MainStack == "unknown" || slices.Contains(c.TargetStacks, x.MainStack)
}

// recentlyAssessed matches the job's company against tracker rows by domain key,
// by name key, or by the tracker's stored name, since ads rarely carry a domain.
func recentlyAssessed(j model.Job, seen []model.SeenCompany, now time.Time, days int) bool {
	key, name, url := Key(j.Company, j.CompanyDomain), Key(j.Company, ""), j.URL
	for _, s := range seen {
		if !Same(s.Key, key) && !Same(s.Key, name) && (s.Name == "" || !Same(Key(s.Name, ""), name)) {
			continue
		}
		if now.Sub(s.LastAssessed) > time.Duration(days)*24*time.Hour {
			return false
		}
		return slices.Contains(s.RoleURLs, url)
	}
	return false
}

func dedupeBatch(jobs []model.Job) []model.Job {
	out := []model.Job{}
	for _, j := range jobs {
		k := Key(j.Company, j.CompanyDomain)
		t := strings.ToLower(strings.TrimSpace(j.Title))
		dup := false
		for i, o := range out {
			if Same(Key(o.Company, o.CompanyDomain), k) && strings.ToLower(strings.TrimSpace(o.Title)) == t {
				if len(j.Desc) > len(o.Desc) {
					out[i] = j
				}
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, j)
		}
	}
	return out
}
