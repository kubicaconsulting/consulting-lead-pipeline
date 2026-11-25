package score

import (
	"math"
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

func near(a, b float64) bool { return math.Abs(a-b) <= 0.1 }

func TestCompanyFintechModernising(t *testing.T) {
	s := Company(model.Assessment{IndustryTier: "fintech", Headcount: 400, Funding: "unknown", Platform: "modernise", Sponsorship: "unknown", LocalEntity: "yes"}, cfg(t))
	if !near(s.CompanyScore, 96.0) || !s.Outreach {
		t.Errorf("score=%v outreach=%v breakdown=%v", s.CompanyScore, s.Outreach, s.Breakdown)
	}
}

func TestCompanyAgency(t *testing.T) {
	s := Company(model.Assessment{IndustryTier: "agency_outsourcing", Headcount: 30, Platform: "maintenance", Sponsorship: "unknown", LocalEntity: "yes"}, cfg(t))
	if !near(s.CompanyScore, 18.0) || s.Outreach {
		t.Errorf("score=%v outreach=%v", s.CompanyScore, s.Outreach)
	}
}

func TestFundingImpliesMaturity(t *testing.T) {
	s := Company(model.Assessment{IndustryTier: "fintech", Funding: "c_plus", Platform: "unknown", Sponsorship: "unknown", LocalEntity: "yes"}, cfg(t))
	if !near(s.Breakdown["maturity"], 26.7) {
		t.Errorf("maturity = %v", s.Breakdown["maturity"])
	}
}

func TestNoLocalEntityBlocksOutreach(t *testing.T) {
	s := Company(model.Assessment{IndustryTier: "fintech", Headcount: 400, Platform: "both", Sponsorship: "offered", LocalEntity: "no"}, cfg(t))
	if s.Outreach || !slices.Contains(s.Flags, "no_local_entity") {
		t.Errorf("outreach=%v flags=%v", s.Outreach, s.Flags)
	}
}

func TestUnknownLocalEntityFlagged(t *testing.T) {
	s := Company(model.Assessment{IndustryTier: "fintech", Headcount: 400, Platform: "both", LocalEntity: "unknown"}, cfg(t))
	if !s.Outreach || !slices.Contains(s.Flags, "local_entity_unknown") {
		t.Errorf("outreach=%v flags=%v", s.Outreach, s.Flags)
	}
}

func TestAllUnknownDoesNotPanic(t *testing.T) {
	s := Company(model.Assessment{}, cfg(t))
	if !near(s.CompanyScore, 40.0) {
		t.Errorf("score = %v, want 40", s.CompanyScore)
	}
}

func TestRole(t *testing.T) {
	c := cfg(t)
	top := model.GateResult{X: model.Extracted{Level: "senior", SeniorSignals: []string{"a", "b", "c", "d"}, SalaryHiYear: 70000, PlatformSignals: []string{"legacy", "platform"}}}
	if r := Role(top, c); !near(r, 100) {
		t.Errorf("top role = %v", r)
	}
	plain := model.GateResult{X: model.Extracted{Level: "senior"}}
	if r := Role(plain, c); !near(r, 53.0) {
		t.Errorf("plain role = %v", r)
	}
	mid := model.GateResult{X: model.Extracted{Level: "lead", SalaryHiYear: 60000, PlatformSignals: []string{"legacy"}}}
	if r := Role(mid, c); !near(r, 40*0.8+30*0.6+30*0.7) {
		t.Errorf("mid role = %v", r)
	}
}

func TestNoSponsorshipBlocksOutreach(t *testing.T) {
	s := Company(model.Assessment{IndustryTier: "fintech", Headcount: 400, Platform: "both", Sponsorship: "none", LocalEntity: "yes"}, cfg(t))
	if s.Outreach || !slices.Contains(s.Flags, "no_sponsorship") {
		t.Errorf("outreach=%v flags=%v", s.Outreach, s.Flags)
	}
}
