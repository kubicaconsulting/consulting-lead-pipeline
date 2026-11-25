// Package extract derives structured fields from a job's text using the market profile's patterns.
package extract

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/config"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

var (
	reCache sync.Map
	yearsRe = regexp.MustCompile(`(?i)(\d{1,2})\s?\+?\s?(?:years|años|yrs|year)`)
)

// re compiles a pattern case-insensitively, once.
func re(p string) *regexp.Regexp {
	if v, ok := reCache.Load(p); ok {
		return v.(*regexp.Regexp)
	}
	r := regexp.MustCompile("(?i)" + p)
	reCache.Store(p, r)
	return r
}

func matchAny(text string, patterns []string) bool {
	for _, p := range patterns {
		if re(p).MatchString(text) {
			return true
		}
	}
	return false
}

func keys(text string, m map[string]string) []string {
	out := []string{}
	for k, p := range m {
		if re(p).MatchString(text) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Run extracts every derived field for one job.
func Run(j model.Job, c *config.Config) model.Extracted {
	text := j.Title + "\n" + j.Desc
	x := model.Extracted{
		Level:           level(j.Title, j.Desc, c),
		City:            city(j, c),
		Mode:            mode(j, c),
		Rights:          "none",
		Sponsorship:     "unknown",
		PlatformSignals: keys(text, c.PlatformSignals),
		SeniorSignals:   keys(j.Desc, c.SeniorSignals),
		Flags:           []string{},
	}
	switch {
	case matchAny(text, c.RightsRequired):
		x.Rights = "required"
	case matchAny(text, c.BasedIn):
		x.Rights = "based_in"
	}
	switch {
	case matchAny(text, c.SponsorRefused):
		x.Sponsorship = "refused"
	case matchAny(text, c.SponsorOffered):
		x.Sponsorship = "offered"
	}
	x.Stacks, x.MainStack = stacks(j.Title, j.Desc, c)

	lo, hi, period := j.SalaryLo, j.SalaryHi, j.SalaryPeriod
	if lo == 0 && hi == 0 {
		lo, hi, period = Salary(j.SalaryText)
		if period == "unknown" {
			lo, hi, period = salaryFromDesc(j.Desc)
		}
	} else if period != "year" && period != "month" && period != "hour" {
		period = guessPeriod(max(lo, hi))
	}
	switch period {
	case "month":
		lo, hi = lo*12, hi*12
		x.Flags = append(x.Flags, "salary_monthly")
	case "hour":
		lo, hi = lo*1760, hi*1760
		x.Flags = append(x.Flags, "salary_hourly")
	}
	x.SalaryLoYear, x.SalaryHiYear = lo, hi
	return x
}

func level(title, desc string, c *config.Config) string {
	switch {
	case matchAny(title, c.EMTitle):
		return "em"
	case matchAny(title, c.JuniorTitle):
		return "junior"
	case matchAny(title, c.LeadTitle):
		return "lead"
	case matchAny(title, c.SeniorTitle):
		return "senior"
	}
	best := 0
	for _, m := range yearsRe.FindAllStringSubmatch(desc, -1) {
		if n, _ := strconv.Atoi(m[1]); n >= 1 && n <= 15 && n > best {
			best = n
		}
	}
	switch {
	case best >= c.SeniorYears && c.SeniorYears > 0:
		return "senior"
	case best > 0:
		return "mid"
	}
	return "unknown"
}

func city(j model.Job, c *config.Config) string {
	loc := strings.ToLower(j.City + " " + j.Location)
	names := make([]string, 0, len(c.MetroAliases))
	for k := range c.MetroAliases {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, alias := range c.MetroAliases[name] {
			if alias == "bcn" || len(alias) <= 3 {
				if re(`\b` + regexp.QuoteMeta(alias) + `\b`).MatchString(loc) {
					return name
				}
				continue
			}
			if strings.Contains(loc, strings.ToLower(alias)) {
				return name
			}
		}
	}
	return ""
}

func mode(j model.Job, c *config.Config) string {
	for _, field := range []string{j.WorkMode, j.City + " " + j.Location, j.Title} {
		switch {
		case matchAny(field, c.HybridWords):
			return "hybrid"
		case matchAny(field, c.RemoteWords):
			return "remote"
		case matchAny(field, c.OnsiteWords):
			return "onsite"
		}
	}
	switch {
	case matchAny(j.Desc, c.HybridWords):
		return "hybrid"
	case matchAny(j.Desc, c.RemoteDescWords):
		return "remote"
	case matchAny(j.Desc, c.OnsiteWords):
		return "onsite"
	}
	return "unknown"
}

func stacks(title, desc string, c *config.Config) ([]string, string) {
	strip := func(s string) string {
		for _, p := range c.StackNoise {
			s = re(p).ReplaceAllString(s, " ")
		}
		return s
	}
	t, d := strip(title), strip(desc)
	all := keys(t+"\n"+d, c.Stacks)
	main := "unknown"
	if hits := keys(t, c.Stacks); len(hits) == 1 {
		main = hits[0]
	} else if len(hits) > 1 {
		main = "mixed"
	} else if len(all) == 1 {
		main = all[0]
	} else if len(all) > 1 {
		target := []string{}
		for _, s := range all {
			if slices.Contains(c.TargetStacks, s) {
				target = append(target, s)
			}
		}
		main = "mixed"
		if len(target) == 0 {
			main = all[0]
		}
	}
	return all, main
}

// guessPeriod reads a board's structured salary with no usable period by its size.
func guessPeriod(v float64) string {
	switch {
	case v > 0 && v <= 500:
		return "hour"
	case v > 500 && v < 15000:
		return "month"
	}
	return "year"
}
