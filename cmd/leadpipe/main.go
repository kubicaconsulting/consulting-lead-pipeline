// Command leadpipe runs the deterministic steps of the lead pipeline over JSON files.
//
//	leadpipe normalize --kind linkedin --in raw.json --out jobs.json
//	leadpipe gate      --in jobs.json --seen seen.json --out-pass passed.json --out-excluded excluded.json
//	leadpipe score     --in assessments.json --out scored.json
//	leadpipe rows      --kind companies|leads --in <file> --out rows.json
//
// Every command takes --market (default es) and --config (default ./config).
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/config"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/gate"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/model"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/score"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/sheet"
	"github.com/kubicaconsulting/consulting-lead-pipeline/internal/source"
)

const usage = "usage: leadpipe normalize|gate|score|rows [flags]  (run 'leadpipe <command> -h' for flags)"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cmds := map[string]func([]string) error{"normalize": normalize, "gate": gateCmd, "score": scoreCmd, "rows": rows}
	fn, ok := cmds[os.Args[1]]
	if !ok {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err := fn(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "leadpipe:", err)
		var ce configError
		if errors.As(err, &ce) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

type configError struct{ error }

func (e configError) Unwrap() error { return e.error }

type common struct {
	fs             *flag.FlagSet
	market, cfgDir string
}

func newFlags(name string) *common {
	c := &common{fs: flag.NewFlagSet(name, flag.ContinueOnError)}
	c.fs.StringVar(&c.market, "market", "es", "market profile code (config/markets/<code>.yaml)")
	c.fs.StringVar(&c.cfgDir, "config", "config", "config directory")
	return c
}

func (c *common) load() (*config.Config, error) {
	cfg, err := config.Load(c.cfgDir, c.market)
	if err != nil {
		return nil, configError{err}
	}
	return cfg, nil
}

func normalize(args []string) error {
	f := newFlags("normalize")
	kind := f.fs.String("kind", "generic", "source kind: linkedin|tecnoempleo|indeed_apify|generic")
	in := f.fs.String("in", "", "raw input JSON")
	out := f.fs.String("out", "jobs.json", "output jobs JSON (appended to if it exists)")
	if err := f.fs.Parse(args); err != nil {
		return err
	}
	if _, err := f.load(); err != nil {
		return err
	}
	raw, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	jobs, err := source.Normalize(*kind, raw, f.market)
	if err != nil {
		return err
	}
	var existing []model.Job
	if b, err := os.ReadFile(*out); err == nil {
		if err := json.Unmarshal(b, &existing); err != nil {
			return fmt.Errorf("existing %s: %w", *out, err)
		}
	}
	return write(*out, append(existing, jobs...))
}

func gateCmd(args []string) error {
	f := newFlags("gate")
	in := f.fs.String("in", "jobs.json", "jobs JSON")
	seenPath := f.fs.String("seen", "", "companies already in the tracker (JSON array of SeenCompany); optional")
	outPass := f.fs.String("out-pass", "passed.json", "jobs that pass every gate")
	outExcl := f.fs.String("out-excluded", "excluded.json", "jobs excluded, with reasons")
	if err := f.fs.Parse(args); err != nil {
		return err
	}
	cfg, err := f.load()
	if err != nil {
		return err
	}
	var jobs []model.Job
	if err := read(*in, &jobs); err != nil {
		return err
	}
	var seen []model.SeenCompany
	if *seenPath != "" {
		if err := read(*seenPath, &seen); err != nil {
			return err
		}
	}
	passed, excluded := gate.Run(jobs, seen, time.Now(), cfg)
	if err := write(*outPass, nonNil(passed)); err != nil {
		return err
	}
	return write(*outExcl, nonNil(excluded))
}

func scoreCmd(args []string) error {
	f := newFlags("score")
	in := f.fs.String("in", "assessments.json", "assessments JSON")
	out := f.fs.String("out", "scored.json", "scored companies JSON")
	if err := f.fs.Parse(args); err != nil {
		return err
	}
	cfg, err := f.load()
	if err != nil {
		return err
	}
	var as []model.Assessment
	if err := read(*in, &as); err != nil {
		return err
	}
	scored := make([]model.Scored, 0, len(as))
	for _, a := range as {
		if a.Market == "" {
			a.Market = f.market
		}
		scored = append(scored, score.Company(a, cfg))
	}
	return write(*out, scored)
}

func rows(args []string) error {
	f := newFlags("rows")
	kind := f.fs.String("kind", "companies", "companies (from scored.json) or leads (from passed.json)")
	in := f.fs.String("in", "", "input JSON")
	out := f.fs.String("out", "rows.json", "rows JSON (array of arrays, sheet column order)")
	if err := f.fs.Parse(args); err != nil {
		return err
	}
	cfg, err := f.load()
	if err != nil {
		return err
	}
	switch *kind {
	case "companies":
		var s []model.Scored
		if err := read(*in, &s); err != nil {
			return err
		}
		return write(*out, sheet.CompanyRows(s, time.Now()))
	case "leads":
		var g []model.GateResult
		if err := read(*in, &g); err != nil {
			return err
		}
		scores := make([]float64, len(g))
		for i := range g {
			scores[i] = score.Role(g[i], cfg)
		}
		return write(*out, sheet.LeadRows(g, scores))
	}
	return fmt.Errorf("unknown rows kind: %s", *kind)
}

func read(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func write(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
