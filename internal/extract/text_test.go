package extract

import (
	"slices"
	"testing"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/config"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

func cfg(t *testing.T) *config.Config {
	t.Helper()
	c, err := config.Load("../../config", "es")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLevel(t *testing.T) {
	c := cfg(t)
	cases := []struct{ title, desc, want string }{
		{"Senior Backend Engineer", "", "senior"},
		{"Backend Engineer (Sénior)", "", "senior"},
		{"Sr. Go Developer", "", "senior"},
		{"Backend Engineer", "You have 5+ years of experience", "senior"},
		{"Backend Engineer", "3 years of experience", "mid"},
		{"Junior Python Developer", "", "junior"},
		{"Staff Engineer", "", "lead"},
		{"Engineering Manager, Payments", "", "em"},
	}
	for _, k := range cases {
		x := Run(model.Job{Title: k.title, Desc: k.desc}, c)
		if x.Level != k.want {
			t.Errorf("Level(%q, %q) = %q, want %q", k.title, k.desc, x.Level, k.want)
		}
	}
}

func TestCityAndMode(t *testing.T) {
	c := cfg(t)
	cases := []struct {
		job        model.Job
		city, mode string
	}{
		{model.Job{City: "Sant Cugat del Vallès, Catalonia"}, "barcelona", "unknown"},
		{model.Job{Location: "Remote - Spain"}, "", "remote"},
		{model.Job{City: "Madrid", Desc: "Hybrid, 3 days in office"}, "madrid", "hybrid"},
		{model.Job{City: "Madrid (Híbrido)", WorkMode: "HYBRID"}, "madrid", "hybrid"},
		{model.Job{City: "Madrid", WorkMode: "Remote Solely"}, "madrid", "remote"},
		{model.Job{City: "España (remoto)"}, "", "remote"},
		{model.Job{City: "BCN", Desc: "This is an on-site role"}, "barcelona", "onsite"},
	}
	for _, k := range cases {
		x := Run(k.job, c)
		if x.City != k.city || x.Mode != k.mode {
			t.Errorf("%+v → city %q mode %q; want %q %q", k.job, x.City, x.Mode, k.city, k.mode)
		}
	}
}

func TestRightsAndSponsorship(t *testing.T) {
	c := cfg(t)
	cases := []struct{ desc, rights, sponsor string }{
		{"You must have the right to work in the EU.", "required", "unknown"},
		{"Debes residir en España.", "based_in", "unknown"},
		{"We offer visa sponsorship and relocation support.", "none", "offered"},
		{"Unfortunately there is no relocation support for this role.", "none", "refused"},
		{"Great team, great coffee.", "none", "unknown"},
	}
	for _, k := range cases {
		x := Run(model.Job{Desc: k.desc}, c)
		if x.Rights != k.rights || x.Sponsorship != k.sponsor {
			t.Errorf("%q → %q/%q; want %q/%q", k.desc, x.Rights, x.Sponsorship, k.rights, k.sponsor)
		}
	}
}

func TestStackAndSignals(t *testing.T) {
	c := cfg(t)
	x := Run(model.Job{Title: "Backend Engineer", Desc: "We use Go and Kafka. We will migrate our legacy monolith to microservices."}, c)
	if !slices.Contains(x.Stacks, "go") || x.MainStack != "go" {
		t.Errorf("stacks %v main %q", x.Stacks, x.MainStack)
	}
	for _, s := range []string{"legacy", "monolith", "migration"} {
		if !slices.Contains(x.PlatformSignals, s) {
			t.Errorf("platform signals %v missing %s", x.PlatformSignals, s)
		}
	}
	y := Run(model.Job{Title: "Java Developer", Desc: "Spring Boot. We go live next month and go beyond expectations."}, c)
	if slices.Contains(y.Stacks, "go") {
		t.Errorf("'go live' read as Go: %v", y.Stacks)
	}
	if y.MainStack != "java" {
		t.Errorf("main stack = %q, want java", y.MainStack)
	}
	z := Run(model.Job{Title: "Senior Python / Go Engineer", Desc: "Python, Go"}, c)
	if z.MainStack != "mixed" {
		t.Errorf("main stack = %q, want mixed", z.MainStack)
	}
}

func TestRunSalaryYearly(t *testing.T) {
	c := cfg(t)
	x := Run(model.Job{SalaryText: "3.500 €/mes"}, c)
	if x.SalaryLoYear != 42000 || x.SalaryHiYear != 42000 || !slices.Contains(x.Flags, "salary_monthly") {
		t.Errorf("monthly → %v-%v flags %v", x.SalaryLoYear, x.SalaryHiYear, x.Flags)
	}
	y := Run(model.Job{SalaryLo: 60000, SalaryHi: 70000, SalaryPeriod: "year"}, c)
	if y.SalaryLoYear != 60000 || y.SalaryHiYear != 70000 {
		t.Errorf("board fields → %v-%v", y.SalaryLoYear, y.SalaryHiYear)
	}
	z := Run(model.Job{Desc: "Salary between 45.000€ and 55.000€ gross/year."}, c)
	if z.SalaryLoYear != 45000 || z.SalaryHiYear != 55000 {
		t.Errorf("desc salary → %v-%v", z.SalaryLoYear, z.SalaryHiYear)
	}
}

func TestStructuredSalaryUnknownPeriod(t *testing.T) {
	c := cfg(t)
	h := Run(model.Job{SalaryLo: 45, SalaryHi: 60, SalaryPeriod: "unknown"}, c)
	if h.SalaryHiYear != 60*1760 || !slices.Contains(h.Flags, "salary_hourly") {
		t.Errorf("hourly guess → %v flags %v", h.SalaryHiYear, h.Flags)
	}
	m := Run(model.Job{SalaryLo: 3500, SalaryHi: 4500}, c)
	if m.SalaryHiYear != 54000 || !slices.Contains(m.Flags, "salary_monthly") {
		t.Errorf("monthly guess → %v flags %v", m.SalaryHiYear, m.Flags)
	}
	y := Run(model.Job{SalaryLo: 60000, SalaryHi: 70000}, c)
	if y.SalaryHiYear != 70000 {
		t.Errorf("yearly guess → %v", y.SalaryHiYear)
	}
	hr := Run(model.Job{SalaryLo: 50, SalaryHi: 70, SalaryPeriod: "hour"}, c)
	if hr.SalaryHiYear != 70*1760 {
		t.Errorf("hour period → %v", hr.SalaryHiYear)
	}
}

func TestMinOnlySalaryText(t *testing.T) {
	lo, hi, p := Salary("A partir de 50.000 €")
	if lo != 50000 || hi != 0 || p != "year" {
		t.Errorf("got %v %v %q", lo, hi, p)
	}
}
