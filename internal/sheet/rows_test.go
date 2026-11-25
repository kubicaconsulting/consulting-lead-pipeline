package sheet

import (
	"testing"
	"time"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
)

var day = time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)

func TestCompanyRowColumns(t *testing.T) {
	rows := CompanyRows([]model.Scored{{
		A:            model.Assessment{Company: "Acme", Market: "es", Domain: "acme.io", OpenRoleURLs: []string{"u1", "u2"}},
		CompanyScore: 95.96, Outreach: true, Flags: []string{"local_entity_unknown"},
	}}, day)
	if len(rows) != 1 || len(rows[0]) != len(CompanyColumns) {
		t.Fatalf("rows=%d cols=%d want %d", len(rows), len(rows[0]), len(CompanyColumns))
	}
	r := rows[0]
	if r[col(CompanyColumns, "company")] != "Acme" || r[col(CompanyColumns, "market")] != "es" {
		t.Errorf("row = %v", r)
	}
	if r[col(CompanyColumns, "company_score")] != 96.0 {
		t.Errorf("score = %v", r[col(CompanyColumns, "company_score")])
	}
	if r[col(CompanyColumns, "open_roles")] != "u1\nu2" || r[col(CompanyColumns, "last_assessed")] != "2026-11-25" {
		t.Errorf("roles/date = %v / %v", r[col(CompanyColumns, "open_roles")], r[col(CompanyColumns, "last_assessed")])
	}
	if r[col(CompanyColumns, "status")] != "new" {
		t.Errorf("status = %v", r[col(CompanyColumns, "status")])
	}
}

func TestLeadRowColumns(t *testing.T) {
	g := model.GateResult{Job: model.Job{Title: "Senior Go", Company: "Acme", Market: "es", URL: "u", Posted: day},
		X: model.Extracted{City: "barcelona", Mode: "hybrid", SalaryLoYear: 60000, SalaryHiYear: 70000, MainStack: "go", Level: "senior"}, Pass: true}
	rows := LeadRows([]model.GateResult{g}, []float64{81.24})
	if len(rows[0]) != len(LeadColumns) {
		t.Fatalf("cols=%d want %d", len(rows[0]), len(LeadColumns))
	}
	if rows[0][col(LeadColumns, "role_score")] != 81.2 || rows[0][col(LeadColumns, "posted")] != "2026-11-25" {
		t.Errorf("row = %v", rows[0])
	}
}

func col(cols []string, name string) int {
	for i, c := range cols {
		if c == name {
			return i
		}
	}
	return -1
}
