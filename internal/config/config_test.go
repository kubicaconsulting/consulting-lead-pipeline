package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadSpainDefault(t *testing.T) {
	c, err := Load("../../config", "es")
	if err != nil {
		t.Fatal(err)
	}
	if c.SalaryFloor != 56000 || c.SalaryTarget != 68000 {
		t.Errorf("salary floor/target = %v/%v", c.SalaryFloor, c.SalaryTarget)
	}
	if c.Currency != "EUR" {
		t.Errorf("currency = %q", c.Currency)
	}
	for _, m := range []string{"onsite", "hybrid", "remote"} {
		if !slices.Contains(c.Cities["barcelona"], m) {
			t.Errorf("barcelona missing mode %s", m)
		}
	}
	if !slices.Equal(c.Cities["*"], []string{"remote"}) {
		t.Errorf("other cities = %v", c.Cities["*"])
	}
	if c.PassThreshold != 65 || c.ReassessDays != 60 || c.DailyTarget != 10 || c.MaxContacts != 3 {
		t.Errorf("thresholds = %v %v %v %v", c.PassThreshold, c.ReassessDays, c.DailyTarget, c.MaxContacts)
	}
}

func TestCompanyWeightsSumTo100(t *testing.T) {
	c, err := Load("../../config", "es")
	if err != nil {
		t.Fatal(err)
	}
	for name, w := range map[string]map[string]float64{"company": c.CompanyWeights, "role": c.RoleWeights} {
		sum := 0.0
		for _, v := range w {
			sum += v
		}
		if sum < 99.8 || sum > 100.2 {
			t.Errorf("%s weights sum = %v", name, sum)
		}
	}
}

func TestMarketOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "markets"), 0o755)
	os.WriteFile(filepath.Join(dir, "global.yaml"), []byte("pass_threshold: 65\nreassess_days: 60\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "markets", "xx.yaml"), []byte("market: xx\npass_threshold: 70\n"), 0o644)
	c, err := Load(dir, "xx")
	if err != nil {
		t.Fatal(err)
	}
	if c.PassThreshold != 70 {
		t.Errorf("pass threshold = %v, want 70", c.PassThreshold)
	}
	if c.ReassessDays != 60 {
		t.Errorf("reassess days = %v, want global 60", c.ReassessDays)
	}
}

func TestUnknownMarket(t *testing.T) {
	_, err := Load("../../config", "zz")
	if err == nil || !strings.Contains(err.Error(), "market profile not found: zz") {
		t.Fatalf("err = %v", err)
	}
}

func TestMarketMapOverrideMergesKeys(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "markets"), 0o755)
	os.WriteFile(filepath.Join(dir, "global.yaml"), []byte("industries: {fintech: 1.0, travel: 0.7}\ncities: {\"*\": [remote]}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "markets", "xx.yaml"), []byte("industries: {travel: 0.2}\ncities: {lisbon: [onsite, hybrid, remote]}\n"), 0o644)
	c, err := Load(dir, "xx")
	if err != nil {
		t.Fatal(err)
	}
	if c.Industries["fintech"] != 1.0 || c.Industries["travel"] != 0.2 {
		t.Errorf("industries = %v", c.Industries)
	}
	if len(c.Cities["*"]) != 1 || len(c.Cities["lisbon"]) != 3 {
		t.Errorf("cities = %v", c.Cities)
	}
}
