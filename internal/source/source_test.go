package source

import (
	"os"
	"strings"
	"testing"
)

func load(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../testdata/raw/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func checkBasics(t *testing.T, kind, file string) {
	t.Helper()
	jobs, err := Normalize(kind, load(t, file), "es")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 3 {
		t.Fatalf("%s: got %d jobs, want 3", kind, len(jobs))
	}
	for i, j := range jobs {
		if j.Title == "" || j.Company == "" || j.URL == "" || j.Desc == "" || j.Posted.IsZero() {
			t.Errorf("%s job %d missing field: %+v", kind, i, j)
		}
		if j.Source != kind || j.Market != "es" {
			t.Errorf("%s job %d source/market = %q/%q", kind, i, j.Source, j.Market)
		}
		if strings.Contains(j.Desc, "</") {
			t.Errorf("%s job %d desc still has HTML", kind, i)
		}
	}
}

func TestNormalizeLinkedIn(t *testing.T) {
	checkBasics(t, "linkedin", "linkedin")
	jobs, _ := Normalize("linkedin", load(t, "linkedin"), "es")
	if jobs[0].Headcount != "10,001+ employees" {
		t.Errorf("headcount = %q", jobs[0].Headcount)
	}
	if jobs[0].City != "Madrid" || jobs[0].WorkMode != "Remote Solely" {
		t.Errorf("city/mode = %q/%q", jobs[0].City, jobs[0].WorkMode)
	}
}

func TestNormalizeTecnoempleo(t *testing.T) {
	checkBasics(t, "tecnoempleo", "tecnoempleo")
	jobs, _ := Normalize("tecnoempleo", load(t, "tecnoempleo"), "es")
	if jobs[0].Company != "lemon.io" || jobs[0].WorkMode != "HYBRID" {
		t.Errorf("company/mode = %q/%q", jobs[0].Company, jobs[0].WorkMode)
	}
}

func TestNormalizeIndeedApify(t *testing.T) {
	checkBasics(t, "indeed_apify", "indeed_apify")
	jobs, _ := Normalize("indeed_apify", load(t, "indeed_apify"), "es")
	if jobs[0].SalaryText != "144.340 € al año" || jobs[0].Headcount != "10,000+" {
		t.Errorf("salary/headcount = %q/%q", jobs[0].SalaryText, jobs[0].Headcount)
	}
}

func TestNormalizeGenericRoundTrip(t *testing.T) {
	jobs, err := Normalize("generic", load(t, "generic"), "es")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].CompanyDomain != "acmepay.io" || jobs[1].Company != "Beta Insure SL" {
		t.Fatalf("got %+v", jobs)
	}
}

func TestNormalizeItemsWrapper(t *testing.T) {
	jobs, err := Normalize("generic", []byte(`{"items":`+string(load(t, "generic"))+`}`), "es")
	if err != nil || len(jobs) != 2 {
		t.Fatalf("jobs=%d err=%v", len(jobs), err)
	}
}

func TestNormalizeUnknownKind(t *testing.T) {
	_, err := Normalize("foo", []byte("[]"), "es")
	if err == nil || err.Error() != "unknown source kind: foo" {
		t.Fatalf("err = %v", err)
	}
}
