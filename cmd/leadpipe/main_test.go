package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "leadpipe")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func run(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, append(args, "--config", "../../config")...)
	out, err := cmd.CombinedOutput()
	code := 0
	if e, ok := err.(*exec.ExitError); ok {
		code = e.ExitCode()
	}
	return string(out), code
}

func count(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatal(err)
	}
	return len(arr)
}

func TestCLIEndToEnd(t *testing.T) {
	bin, dir := build(t), t.TempDir()
	p := func(n string) string { return filepath.Join(dir, n) }
	os.WriteFile(p("seen.json"), []byte("[]"), 0o644)

	steps := [][]string{
		{"normalize", "--kind", "generic", "--in", "../../testdata/raw/generic.json", "--out", p("jobs.json")},
		{"gate", "--in", p("jobs.json"), "--seen", p("seen.json"), "--out-pass", p("passed.json"), "--out-excluded", p("excluded.json")},
		{"score", "--in", "../../testdata/assessments.json", "--out", p("scored.json")},
		{"rows", "--kind", "companies", "--in", p("scored.json"), "--out", p("company_rows.json")},
		{"rows", "--kind", "leads", "--in", p("passed.json"), "--out", p("lead_rows.json")},
	}
	for _, s := range steps {
		if out, code := run(t, bin, s...); code != 0 {
			t.Fatalf("%v: exit %d\n%s", s, code, out)
		}
	}
	if n := count(t, p("jobs.json")); n != 2 {
		t.Errorf("jobs = %d, want 2", n)
	}
	if n := count(t, p("passed.json")) + count(t, p("excluded.json")); n != 2 {
		t.Errorf("gated = %d, want 2", n)
	}
	if n := count(t, p("company_rows.json")); n != 2 {
		t.Errorf("company rows = %d, want 2", n)
	}
	if count(t, p("lead_rows.json")) != count(t, p("passed.json")) {
		t.Errorf("lead rows != passed jobs")
	}
}

func TestCLINormalizeAppends(t *testing.T) {
	bin, dir := build(t), t.TempDir()
	out := filepath.Join(dir, "jobs.json")
	for i := 0; i < 2; i++ {
		if o, code := run(t, bin, "normalize", "--kind", "generic", "--in", "../../testdata/raw/generic.json", "--out", out); code != 0 {
			t.Fatal(o)
		}
	}
	if n := count(t, out); n != 4 {
		t.Errorf("jobs after two appends = %d, want 4", n)
	}
}

func TestCLIUnknownMarket(t *testing.T) {
	bin := build(t)
	out, code := run(t, bin, "score", "--market", "zz", "--in", "../../testdata/assessments.json", "--out", filepath.Join(t.TempDir(), "x.json"))
	if code != 2 || !strings.Contains(out, "market profile not found") {
		t.Errorf("exit %d, out %q", code, out)
	}
}
