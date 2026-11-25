package gate

import (
	"slices"
	"testing"
	"time"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/config"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

var now = time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)

func cfg(t *testing.T) *config.Config {
	t.Helper()
	c, err := config.Load("../../config", "es")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// base is a job that passes every gate: senior Go role in Barcelona, hybrid.
func base() model.Job {
	return model.Job{
		ID: "1", Title: "Senior Backend Engineer", Company: "Acme", URL: "https://x/1",
		City: "Barcelona", Desc: "We build our platform in Go. Hybrid, 2 days in office.",
	}
}

func one(t *testing.T, j model.Job, seen []model.SeenCompany) model.GateResult {
	t.Helper()
	p, e := Run([]model.Job{j}, seen, now, cfg(t))
	if len(p)+len(e) != 1 {
		t.Fatalf("got %d passed + %d excluded", len(p), len(e))
	}
	if len(p) == 1 {
		return p[0]
	}
	return e[0]
}

func TestGates(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*model.Job)
		pass   bool
		reason string // expected in Reasons when excluded
		flag   string // expected in Flags when passed
	}{
		{"base passes, salary unknown flagged", func(j *model.Job) {}, true, "", "salary_unknown"},
		{"salary below floor", func(j *model.Job) { j.SalaryText = "45.000 - 50.000 €" }, false, "salary_below_floor", ""},
		{"salary between floor and target", func(j *model.Job) { j.SalaryText = "60.000 - 65.000 €" }, true, "", "salary_below_target"},
		{"madrid hybrid excluded", func(j *model.Job) { j.City = "Madrid" }, false, "location_mode", ""},
		{"valencia remote passes", func(j *model.Job) { j.City = "Valencia"; j.WorkMode = "Remote"; j.Desc = "Go platform." }, true, "", ""},
		{"rights required", func(j *model.Job) { j.Desc += " You must have the right to work in the EU." }, false, "requires_work_rights", ""},
		{"based in flagged", func(j *model.Job) { j.Desc += " Debes residir en España." }, true, "", "based_in_wording"},
		{"junior excluded", func(j *model.Job) { j.Title = "Junior Backend Engineer" }, false, "seniority", ""},
		{"java only excluded", func(j *model.Job) { j.Title = "Senior Java Engineer"; j.Desc = "Spring Boot. Hybrid." }, false, "stack", ""},
		{"senior title but 3+ years excluded", func(j *model.Job) { j.Desc += " 3+ years of experience." }, false, "experience_below_min", ""},
		{"10+ years flagged as preferred", func(j *model.Job) { j.Desc += " 10+ years of experience." }, true, "", "experience_10_plus"},
		{"unknown level flagged", func(j *model.Job) { j.Title = "Backend Engineer" }, true, "", "level_unknown"},
	}
	for _, k := range cases {
		j := base()
		k.mutate(&j)
		g := one(t, j, nil)
		if g.Pass != k.pass {
			t.Errorf("%s: pass=%v reasons=%v", k.name, g.Pass, g.Reasons)
			continue
		}
		if k.reason != "" && !slices.Contains(g.Reasons, k.reason) {
			t.Errorf("%s: reasons=%v, want %s", k.name, g.Reasons, k.reason)
		}
		if k.flag != "" && !slices.Contains(g.Flags, k.flag) {
			t.Errorf("%s: flags=%v, want %s", k.name, g.Flags, k.flag)
		}
	}
}

func TestRecentlyAssessed(t *testing.T) {
	j := base()
	seen := func(days int, urls ...string) []model.SeenCompany {
		return []model.SeenCompany{{Key: "acme", LastAssessed: now.AddDate(0, 0, -days), RoleURLs: urls}}
	}
	if g := one(t, j, seen(10, j.URL)); g.Pass || !slices.Contains(g.Reasons, "recently_assessed") {
		t.Errorf("same role 10 days ago: pass=%v reasons=%v", g.Pass, g.Reasons)
	}
	if g := one(t, j, seen(10, "https://x/old")); !g.Pass {
		t.Errorf("new role 10 days ago should pass: %v", g.Reasons)
	}
	if g := one(t, j, seen(61, j.URL)); !g.Pass {
		t.Errorf("61 days ago should pass: %v", g.Reasons)
	}
}

func TestKey(t *testing.T) {
	if a, b := Key("Glovoapp23 SL", ""), Key("GLOVO", ""); !Same(a, b) || b != "glovo" {
		t.Errorf("Key = %q / %q, want the same company", a, b)
	}
	if Same(Key("Acme", ""), Key("Acmeco Bank", "")) == false {
		t.Errorf("prefix names should match")
	}
	if Same(Key("Bit", ""), Key("Bitpanda", "")) {
		t.Errorf("short prefixes (<4 chars) must not match")
	}
	if k := Key("Acme", "acme.io"); k != "acme.io" {
		t.Errorf("Key with domain = %q", k)
	}
	if k := Key("Beta Insure S.L.", ""); k != "betainsure" {
		t.Errorf("Key = %q, want betainsure", k)
	}
	if k := Key("Acme", "https://www.Acme.io/careers"); k != "acme.io" {
		t.Errorf("Key from URL = %q", k)
	}
}

func TestBatchDedupeKeepsRichest(t *testing.T) {
	a, b := base(), base()
	b.ID, b.URL, b.Desc = "2", "https://x/2", b.Desc+" Plus a much longer description of the role."
	p, _ := Run([]model.Job{a, b}, nil, now, cfg(t))
	if len(p) != 1 || p[0].Job.ID != "2" {
		t.Fatalf("passed=%d first=%v", len(p), p)
	}
}

func TestSeenByDomainMatchesJobByName(t *testing.T) {
	j := base()
	j.Company = "Glovo"
	seen := []model.SeenCompany{{Key: "glovoapp.com", Name: "Glovo", LastAssessed: now.AddDate(0, 0, -5), RoleURLs: []string{j.URL}}}
	if g := one(t, j, seen); g.Pass || !slices.Contains(g.Reasons, "recently_assessed") {
		t.Errorf("company seen by domain not matched by name: pass=%v reasons=%v", g.Pass, g.Reasons)
	}
}

func TestAdRefusingSponsorshipExcluded(t *testing.T) {
	j := base()
	j.Desc += " No visa sponsorship available."
	if g := one(t, j, nil); g.Pass || !slices.Contains(g.Reasons, "sponsorship_refused") {
		t.Errorf("pass=%v reasons=%v", g.Pass, g.Reasons)
	}
}

func TestMinimumOnlySalaryNotExcluded(t *testing.T) {
	j := base()
	j.SalaryLo = 50000
	if g := one(t, j, nil); !g.Pass || !slices.Contains(g.Flags, "salary_min_only") {
		t.Errorf("board min-only: pass=%v reasons=%v flags=%v", g.Pass, g.Reasons, g.Flags)
	}
	k := base()
	k.SalaryText = "Desde 50.000€ brutos/año"
	if g := one(t, k, nil); !g.Pass || !slices.Contains(g.Flags, "salary_min_only") {
		t.Errorf("text min-only: pass=%v reasons=%v flags=%v", g.Pass, g.Reasons, g.Flags)
	}
}
