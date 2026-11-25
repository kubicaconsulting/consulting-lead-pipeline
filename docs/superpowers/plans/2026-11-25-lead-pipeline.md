# Consulting Lead Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A daily, market-configurable pipeline that collects job ads and companies, applies hard gates, scores companies and roles, keeps a Google Sheet tracker, and leaves Gmail drafts for manual review.

**Architecture:** A Go CLI (`leadpipe`) does every deterministic step over JSON files: normalise raw source data into one `Job` shape, extract fields, gate and dedupe, and turn Claude's structured company assessments into scores and sheet rows. A daily Claude scheduled task follows `RUNBOOK.md`: it pulls data through the connectors (Indeed, Apify), runs `leadpipe`, does the research and judgement steps with the prompts in `prompts/`, then writes rows through the Sheets connector and drafts through Gmail. Market rules live in `config/markets/<code>.yaml`; Spain (`es`) is the default.

**Tech Stack:** Go 1.24, standard library + `gopkg.in/yaml.v3`; Google Sheets, Gmail, Drive, Indeed and Apify connectors (used by the runner, not by Go).

**Spec:** `docs/specs/2026-11-25-lead-qualification-pipeline-design.md` (read it before starting). Reference implementation for extraction rules: `/home/claude/bench2/{normalize,filter,extract}.py` and raw samples in `/home/claude/bench2/raw/`.

## Global Constraints

- Never send email. Only `create_draft`. No code path or runbook step calls a send tool.
- Hard gates (Spain): salary floor **€56,000 gross/yr** if stated; salary target **€68,000 gross/yr**; Barcelona: onsite, hybrid or remote; any other city: remote only; exclude ads requiring existing EU/Spanish work rights; "based in Spain" alone = pass + flag; exclude junior/mid titles; main stack must include Python, Go or Node.js/TypeScript.
- Company score weights (sum 100): industry 33.3, maturity 26.7, platform 33.3, sponsorship 6.7 (spec §4.2 company dimensions reweighted). Role score weights: level & scope 40, salary 30, platform signals 30.
- Outreach pass threshold: company score ≥ **65**. Companies without a local legal entity: never outreach.
- Re-assessment window: **60 days** unless a new matching role appears. Daily target: **10** fully assessed companies. Contacts: 1–3 per passing company.
- Store only work-context contact data: name, title, LinkedIn URL, work email, email status (`found` | `unverified`), source.
- Every market-specific value comes from the market profile; no Spain literals in Go code.
- No secrets in the repo.

## Review Focus

1. Salary written as monthly, `45.000 €`, `45k`, `45.000-55.000€ brutos/año`, or a range with no keyword → normalised to gross yearly EUR; monthly ×12 with a `salary_monthly` flag. (Task 3)
2. One company under different names (`Glovo` / `Glovoapp23 SL` / `GLOVO`) or with/without domain → one dedupe key (domain first, else name stripped of legal suffixes). (Task 4)
3. City variants (`Barcelona, Catalonia`, `Sant Cugat del Vallès`, `BCN`, `Remote - Spain`, `España (remoto)`) → mapped through the profile's metro list and remote words. (Task 3)
4. Seniority not in the title (`Backend Engineer` with `5+ years`) or Spanish forms (`Sénior`, `Sr.`) → senior. (Task 3)
5. An assessment file with a missing or unknown field (e.g. headcount unknown) → scored with the "unknown" value, never a crash or a zero score. (Task 5)

---

## File structure

```
cmd/leadpipe/main.go            CLI: normalize | gate | score | rows
internal/config/config.go       Global + market profile loading, merge
internal/model/model.go         Job, Extracted, GateResult, Assessment, Scored, SeenCompany
internal/source/*.go            One normaliser per source → []model.Job
internal/extract/*.go           Salary, stack, level, city/mode, rights/sponsorship, platform & senior signals
internal/gate/gate.go           Hard gates + dedupe
internal/score/score.go         Company and role scores
internal/sheet/rows.go          Scored → sheet row arrays (column order fixed in docs/sheet-schema.md)
config/global.yaml              Weights, thresholds, industry tiers, seniority patterns
config/markets/es.yaml          Spain profile (default)
prompts/{assess_company,find_contacts,draft_email}.md
RUNBOOK.md, README.md, docs/sheet-schema.md
testdata/                       Small fixtures sampled from bench2/raw
```

---

### Task 1: Module, model and config loading

**Files:**
- Create: `go.mod` (module `github.com/kubicaconsulting/consulting-lead-pipeline`), `internal/model/model.go`, `internal/config/config.go`, `config/global.yaml`, `config/markets/es.yaml`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces:
  - `config.Load(dir string, market string) (*config.Config, error)`. Merges `global.yaml` with `markets/<market>.yaml`; market values override global ones.
  - `config.Config` fields: `Market`, `Country`, `Currency`, `SalaryFloor`, `SalaryTarget float64`, `Cities map[string][]string` (canonical city → allowed modes; key `"*"` = any other city), `MetroAliases map[string][]string`, `RemoteWords []string`, `RightsRequired`, `BasedIn`, `SponsorOffered`, `SponsorRefused []string` (regexes), `JuniorTitle`, `SeniorTitle`, `LeadTitle`, `EMTitle []string`, `Stacks map[string]string` (stack → regex), `PlatformSignals`, `SeniorSignals map[string]string`, `Industries map[string]float64` (tier key → sub-score), `CompanyWeights`, `RoleWeights map[string]float64`, `PassThreshold float64`, `ReassessDays`, `DailyTarget`, `MaxContacts int`, `Tier string` (free|paid), `Sources []SourceCfg`, `DiscoveryQueries []string`, `OutreachLang string`.
  - Model types (used by every later task):
    ```go
    type Job struct {
        ID, Source, Market, Title, Company, CompanyDomain, City, Location, URL, Lang, Desc string
        Posted time.Time
        SalaryText string; SalaryLo, SalaryHi float64; SalaryPeriod string // year|month|unknown
        Headcount, Industry string
    }
    type Extracted struct {
        Stacks []string; MainStack string
        Level string            // junior|mid|senior|lead|em|unknown
        Mode string             // onsite|hybrid|remote|unknown
        City string             // canonical
        Rights string           // required|based_in|none
        Sponsorship string      // offered|refused|unknown
        SalaryLoYear, SalaryHiYear float64
        PlatformSignals, SeniorSignals, Flags []string
    }
    type GateResult struct { Job Job; X Extracted; Pass bool; Reasons, Flags []string; DedupeKey string }
    type Assessment struct {
        Company, Domain, Market, Website string
        IndustryTier string          // key in Config.Industries, or "unknown"
        Headcount int                // 0 = unknown
        Funding string               // seed|a|b|c_plus|public|group|bootstrapped|unknown
        Platform string              // scale|modernise|both|product|maintenance|unknown
        Sponsorship string           // offered|prior|unknown|none
        LocalEntity string           // yes|no|unknown
        OpenRoleURLs []string; Reason, Sources string
    }
    type Scored struct { A Assessment; CompanyScore float64; Breakdown map[string]float64; Outreach bool; Flags []string }
    type SeenCompany struct { Key string; LastAssessed time.Time; RoleURLs []string }
    ```

- [ ] **Step 1: Write failing tests**
  - `TestLoadSpainDefault`: `Load("../../config","es")` → `SalaryFloor==56000`, `SalaryTarget==68000`, `Currency=="EUR"`, `Cities["barcelona"]` contains `onsite`,`hybrid`,`remote`, `Cities["*"] == []string{"remote"}`, `PassThreshold==65`, `ReassessDays==60`, `DailyTarget==10`, `MaxContacts==3`.
  - `TestCompanyWeightsSumTo100`: sum of `CompanyWeights` within 0.2 of 100; same for `RoleWeights`.
  - `TestMarketOverridesGlobal`: temp dir with global `PassThreshold: 65` and market `pass_threshold: 70` → 70.
  - `TestUnknownMarket`: `Load(dir,"zz")` → error containing `"market profile not found: zz"`.
- [ ] **Step 2:** `go test ./internal/config/...` → FAIL (package missing).
- [ ] **Step 3: Write `config/global.yaml`, `config/markets/es.yaml`, the model types and `config.Load`.** Values:
  - Industries: `fintech: 1.0`, `insurtech|travel|mobility|energy|healthtech: 0.7`, `other_product: 0.4`, `agency_outsourcing: 0.1`, `unknown: 0.4`.
  - Spain regexes in English and Spanish. Rights-required examples: `right to work in (spain|the eu)`, `eu (work )?(permit|authori[sz]ation)`, `permiso de trabajo (en vigor|vigente)`, `nie (en vigor|vigente)`. Refused sponsorship: `no (visa )?sponsorship`, `unable to sponsor`, `no ofrecemos (patrocinio|visado)`. Based-in: `based in spain`, `residir en españa`, `residencia en españa`. Offered sponsorship: `visa sponsorship`, `relocation (package|support|assistance)`, `(ayuda|paquete) de (reubicación|relocalización)`.
  - Barcelona metro aliases: `bcn`, `sant cugat`, `l'hospitalet`, `badalona`, `castelldefels`, `sant joan despí`, `cornellà`, `el prat`.
  - Remote words: `remote`, `remoto`, `teletrabajo`, `100% remote`.
  - Port stack, title, platform and senior-signal patterns from `bench2/filter.py` (`STACK`, `REL_TITLE`, `EXCL_TITLE`, `NOFIT_TITLE`) and `bench2/extract.py` (`SENIOR`). Platform signals: `platform`, `high[- ]traffic`, `millions of (users|requests|events)`, `distributed systems`, `legacy`, `migrat`, `re-?architect`, `monolith`, `replatform`, `moderni[sz]`.
  - Sources: Indeed (connector, `country: es`), Apify `fantastic-jobs/advanced-linkedin-job-search-api`, `blackfalcondata/tecnoempleo-scraper`, each with its input template.
  - Discovery queries: Spanish fintech/scale-up funding news, Barcelona tech ecosystem lists (`"fintech Barcelona series B"`, `"scale-up Barcelona hiring backend"`, `"fintech España ronda de financiación 2026"`).
- [ ] **Step 4:** `go test ./internal/config/...` → PASS.
- [ ] **Step 5:** Commit: `feat: module, model types and market-profile config (es default)`.

### Task 2: Source normalisers

**Files:**
- Create: `internal/source/source.go`, `internal/source/linkedin.go`, `internal/source/tecnoempleo.go`, `internal/source/indeed.go`, `internal/source/generic.go`
- Create fixtures: `testdata/raw/linkedin.json`, `testdata/raw/tecnoempleo.json`, `testdata/raw/indeed_apify.json` (3 records each, copied from `bench2/raw/li1.json`, `te1.json`, `in1.json`), `testdata/raw/generic.json` (2 hand-written `Job` records)
- Test: `internal/source/source_test.go`

**Interfaces:**
- Consumes: `model.Job`.
- Produces: `source.Normalize(kind string, raw []byte, market string) ([]model.Job, error)`. `kind` is one of `linkedin`, `tecnoempleo`, `indeed_apify`, `generic`. The `generic` kind is the shape the runner writes from Indeed connector results or manual finds: a JSON array of `Job`.

- [ ] **Step 1: Failing tests:**
  - `TestNormalizeLinkedIn`: 3 jobs; each has a non-empty `Title`, `Company`, `URL`, `Desc` and a non-zero `Posted`; `Source=="linkedin"`; `Market=="es"`. The first record's `Headcount` matches the fixture's employee-count field.
  - Same pattern for `tecnoempleo` and `indeed_apify`.
  - `TestNormalizeGenericRoundTrip`: unmarshalled values equal the fixture.
  - `TestNormalizeUnknownKind`: error `"unknown source kind: foo"`.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement field mappings ported from `bench2/normalize.py`, one function per kind. Use the same field names that script reads. HTML is stripped from `Desc` as `clean()` does.
- [ ] **Step 4:** `go test ./internal/source/...` → PASS.
- [ ] **Step 5:** Commit: `feat: source normalisers for linkedin, tecnoempleo, indeed and generic`.

### Task 3: Field extraction

**Files:**
- Create: `internal/extract/salary.go`, `internal/extract/text.go` (stack, level, city/mode, rights/sponsorship, signals), `internal/extract/extract.go`
- Test: `internal/extract/salary_test.go`, `internal/extract/text_test.go`

**Interfaces:**
- Consumes: `model.Job`, `*config.Config`.
- Produces:
  - `extract.Run(j model.Job, c *config.Config) model.Extracted`
  - `extract.Salary(text string) (lo, hi float64, period string)`

- [ ] **Step 1: Failing table tests** (exact cases):
  - Salary:
    - `"Salary between 45.000€ and 55.000€ gross/year"` → 45000, 55000, year.
    - `"up to 60k"` → 0, 60000, year.
    - `"140,000 185,000 USD"` → 140000, 185000, year.
    - `"45.000-55.000€ brutos/año"` → 45000, 55000, year.
    - `"3.500 €/mes"` → 3500, 3500, month. In `Run` this becomes 42000/yr with flag `salary_monthly`.
    - `"competitive salary"` → 0, 0, unknown.
    - A 4-digit year such as `"since 2019"` is never read as a salary.
  - Level:
    - `"Senior Backend Engineer"` → senior.
    - `"Backend Engineer (Sénior)"` → senior.
    - `"Sr. Go Developer"` → senior.
    - `"Backend Engineer"` with desc `"5+ years of experience"` → senior.
    - `"Junior Python Developer"` → junior.
    - `"Staff Engineer"` → lead.
    - `"Engineering Manager"` → em.
  - City and mode:
    - Location `"Sant Cugat del Vallès, Catalonia"` → city barcelona.
    - Location `"Remote - Spain"` → mode remote.
    - Location `"Madrid"` with desc `"hybrid, 3 days in office"` → madrid, hybrid.
  - Rights and sponsorship:
    - `"You must have the right to work in the EU"` → Rights required.
    - `"Debes residir en España"` → based_in.
    - `"We offer visa sponsorship and relocation support"` → Sponsorship offered.
    - `"no relocation support"` (IKEA case) → Sponsorship refused.
  - Stack: desc mentioning Go and Kafka only → Stacks contains `go`, MainStack `go`.
  - Signals: desc `"migrate our legacy monolith to microservices"` → PlatformSignals contains `legacy` and `monolith`.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement. Port the salary keyword-window approach from `bench2/extract.py::salary`, the stack logic from `classify_stack` and the level logic from `seniority`/`years`. Every regex comes from config. Number parsing accepts `.` and `,` as thousands separators and a `k` suffix. Treat 1,000–9,999 as monthly only when a month word (`mes|month|mensual`) is present.
- [ ] **Step 4:** `go test ./internal/extract/...` → PASS.
- [ ] **Step 5:** Commit: `feat: field extraction (salary, level, city/mode, rights, stack, signals)`.

### Task 4: Gates and dedupe

**Files:**
- Create: `internal/gate/gate.go`
- Test: `internal/gate/gate_test.go`

**Interfaces:**
- Consumes: `extract.Run`, `model.SeenCompany`, `*config.Config`.
- Produces:
  - `gate.Key(company, domain string) string`: the lowercased domain when present; otherwise the lowercased name with legal suffixes removed (`sl`, `s.l.`, `sa`, `s.a.`, `slu`, `inc`, `ltd`, `gmbh`, `bv`), digits and punctuation.
  - `gate.Run(jobs []model.Job, seen []model.SeenCompany, now time.Time, c *config.Config) (passed, excluded []model.GateResult)`

- [ ] **Step 1: Failing tests.** One job per case; check `Pass` and that `Reasons` contains the string named.
  - Stated salary hi 50000 → excluded, `salary_below_floor`.
  - Salary unknown → pass, flag `salary_unknown`.
  - Salary 60000–65000 → pass, flag `salary_below_target`.
  - Madrid hybrid → excluded, `location_mode`.
  - Barcelona hybrid → pass.
  - Valencia remote → pass.
  - Rights required → excluded, `requires_work_rights`.
  - Based-in → pass, flag `based_in_wording`.
  - Junior → excluded, `seniority`.
  - Java-only stack → excluded, `stack`.
  - Company seen 10 days ago with the same role URL → excluded, `recently_assessed`.
  - Seen 10 days ago with a new role URL → pass.
  - Seen 61 days ago → pass.
  - `Key("Glovoapp23 SL","")` equals `Key("GLOVO","")`, and both equal `"glovo"`.
  - `Key("Acme","acme.io")` equals `"acme.io"`.
  - Two jobs with the same key and title in one batch → one passed (the one with the longer `Desc`).
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement. Gate order: rights, seniority, stack, location/mode, salary, recently assessed. The first failure sets the reason; flags still collect.
- [ ] **Step 4:** `go test ./internal/gate/...` → PASS.
- [ ] **Step 5:** Commit: `feat: hard gates and company dedupe`.

### Task 5: Scoring

**Files:**
- Create: `internal/score/score.go`
- Test: `internal/score/score_test.go`

**Interfaces:**
- Consumes: `model.Assessment`, `model.GateResult`, `*config.Config`.
- Produces:
  - `score.Company(a model.Assessment, c *config.Config) model.Scored`
  - `score.Role(g model.GateResult, c *config.Config) float64`

- [ ] **Step 1: Failing tests.** Use the exact sub-score table in Step 3.
  - Fintech, headcount 400, funding unknown, platform modernise, sponsorship unknown, LocalEntity yes → `33.3 + 26.7 + 33.3 + 6.7*0.4` = 96.0 (±0.1); `Outreach==true`.
  - Agency, headcount 30, platform maintenance, sponsorship unknown → `3.33+5.34+6.66+2.68` = 18.0 (±0.1); `Outreach==false`.
  - Fintech, headcount 0, funding `c_plus` → maturity 1.0.
  - LocalEntity `no` → `Outreach==false` with flag `no_local_entity`, whatever the score.
  - LocalEntity `unknown` → outreach allowed, flag `local_entity_unknown`.
  - An all-unknown assessment → no panic; company score = 0.4 × 100 = 40.0.
  - Role score: senior with 4 senior signals, salary ≥ target, 2 platform signals → 100. Senior with 0 signals, salary unknown, 0 platform signals → `40*0.8 + 30*0.4 + 30*0.3` = 53.0.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement with these sub-scores (0–1):

  | Dimension | Values |
  |---|---|
  | Industry | `Config.Industries[tier]` (unknown 0.4) |
  | Maturity | 1.0 if headcount ≥200 or funding ∈ {c_plus, public, group}; 0.6 if 50–199 or funding b; 0.2 if 1–49; 0.4 if unknown |
  | Platform | scale, modernise or both: 1.0; product: 0.5; maintenance: 0.2; unknown: 0.4 |
  | Sponsorship | offered: 1.0; prior: 0.8; unknown: 0.4; none: 0.0 |
  | Level | senior or lead: 0.8; em: 0.65; plus 0.05 per senior signal, max +0.2 |
  | Salary | ≥ target: 1.0; floor to target: 0.6; unknown: 0.4 |
  | Platform signals | 0: 0.3; 1: 0.7; ≥2: 1.0 |

  Score = Σ weight × sub-score. `Breakdown` holds each weighted part.
- [ ] **Step 4:** `go test ./internal/score/...` → PASS.
- [ ] **Step 5:** Commit: `feat: company and role scoring`.

### Task 6: CLI and sheet rows

**Files:**
- Create: `cmd/leadpipe/main.go`, `internal/sheet/rows.go`, `docs/sheet-schema.md`
- Test: `internal/sheet/rows_test.go`, `cmd/leadpipe/main_test.go`

**Interfaces:**
- Consumes: all earlier packages.
- Produces the CLI (all I/O is JSON files):
  - `leadpipe normalize --market es --kind linkedin --in raw.json --out jobs.json` (appends to `--out` if it exists)
  - `leadpipe gate --market es --in jobs.json --seen seen.json --out-pass passed.json --out-excluded excluded.json`
  - `leadpipe score --market es --in assessments.json --out scored.json`
  - `leadpipe rows --kind companies|leads --in <file> --out rows.json`
  - `--config` defaults to `./config`; `--market` defaults to `es`.
  - `sheet.CompanyRows([]model.Scored) [][]any` and `sheet.LeadRows([]model.GateResult, roleScores []float64) [][]any`. Column order is exactly as in `docs/sheet-schema.md`.

- [ ] **Step 1: Failing tests:**
  - `TestCompanyRowColumns`: first row length equals the number of Companies columns in the schema; column 0 is company name; the score column is rounded to 1 decimal.
  - `TestCLIEndToEnd`: builds the binary and runs `normalize` (generic fixture) → `gate` → `score` (fixture `testdata/assessments.json` with 2 assessments) → `rows`. Asserts exit 0 and expected row counts.
  - `TestCLIUnknownMarket`: exit code 2, stderr contains `market profile not found`.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Write `docs/sheet-schema.md`:
  - Tabs: Companies, Contacts, Leads, Config, Run log, Dashboard.
  - Column lists follow spec §6.2, plus a `market` column on Companies, Contacts and Leads.
  - Columns marked *manual* are never written by the pipeline: `status`, `notes`, `opt_out` and Contacts `outreach_status` beyond `drafted`/`contacted`/`replied`.
  - Dashboard formulas: COUNTIFS by status and market; average score; top 10 companies by score in the last 7 days via SORT/FILTER on the Companies tab. These are Google Sheets formulas.

  Then implement the CLI with the `flag` package and the row builders.
- [ ] **Step 4:** `go test ./...` → PASS; `go vet ./...` clean.
- [ ] **Step 5:** Commit: `feat: leadpipe CLI and sheet row builders`.

### Task 7: Runbook, prompts and tracker setup (first live run)

**Files:**
- Create: `RUNBOOK.md`, `prompts/assess_company.md`, `prompts/find_contacts.md`, `prompts/draft_email.md`, `README.md`
- External, done through connectors and recorded in `README.md`:
  - Drive folder "Spain Job Search";
  - native Google Sheet "Lead Tracker" with the tabs and headers from `docs/sheet-schema.md`;
  - the IDs of the folder, the sheet and the CV v3 PDFs.

**Interfaces:**
- Consumes: the `leadpipe` CLI, `docs/sheet-schema.md`, CV v3 PDF links:
  - Senior: `1N2wIuHBjSj_CWGlKLlPzavGfWmdWdgV5`
  - Lead: `187A0pnK8mw_G0NvjERXyR9PylAD7BLe3`
  - Architecture: `1-ZMg0Smp1eXFdHyuBD1MPP6vwdoUkgQ5`
- Produces: `RUNBOOK.md`, which the scheduled task follows verbatim.

- [ ] **Step 1: Write `RUNBOOK.md`** with numbered steps. Each step names the exact tool or command and its input and output file.
  1. Clone the repo; `go build -o leadpipe ./cmd/leadpipe`.
  2. Read the Companies tab into `state/seen.json`.
  3. Pull the Indeed connector results for each profile query and write them as `generic` jobs. Pull Apify actors only if `tier: paid` or the free credit remains (check run cost); write the raw datasets.
  4. Run `leadpipe normalize` for each source, then `leadpipe gate`.
  5. Build the candidate company list: companies from passed jobs plus discovery queries. Top up to `DailyTarget` with discovery.
  6. For each company, follow `prompts/assess_company.md`. Output must be one `Assessment` JSON object with every field. Use `unknown` rather than guessing, and cite sources in `Sources`.
  7. Run `leadpipe score`, then `leadpipe rows`. Append to the Companies and Leads tabs. Update `last_assessed`.
  8. For each `Outreach==true` company, follow `prompts/find_contacts.md` (max `MaxContacts`; GDPR data limits). Skip any contact already in Contacts or with `opt_out`.
  9. For each new contact, follow `prompts/draft_email.md`, then `create_draft`. **Never send.** Write the draft link and `drafted` status to Contacts.
  10. Search Gmail Sent and Inbox for contacted addresses and update `contacted` / `replied`.
  11. Append a Run log row; post the summary (assessed count, passes with reasons, draft links, errors).
- [ ] **Step 2: Write the three prompts.**
  - `assess_company`: rubric values exactly as in the `Assessment` enums and Task 5's table; evidence required for headcount, funding and local entity.
  - `find_contacts`: target roles by company size (spec §6.1 step 6); `unverified` when an email is inferred from a pattern.
  - `draft_email`: written in `OutreachLang`. Under 150 words. One specific detail about the company's platform or legacy work. CV variant: Architecture if platform is modernise/both, Lead if the contact is CTO/VP Engineering at a company under 200 people, otherwise Senior. Link the PDF. Apply the humanizer patterns: no dashes, no not-X-but-Y, no triads for rhythm. No invented facts about Mossaab; source them from the CV v3 text only.
- [ ] **Step 3: Create the Drive folder and Sheet** through the Drive/Sheets connectors. Put the headers and Dashboard formulas in, then record the IDs in `README.md`.
- [ ] **Step 4: First live run, attended.** Follow `RUNBOOK.md` end to end. Pass criteria:
  - ≥10 companies assessed, or the shortfall explained in Run log;
  - rows visible in the Sheet;
  - ≥1 Gmail draft created and **not sent**;
  - Dashboard formulas show no errors.
- [ ] **Step 5:** Commit: `docs: runbook, prompts, tracker ids; first live run`.

### Task 8: Schedule the daily run

**Files:**
- Modify: `README.md` (schedule, how to change market/tier, how to pause)

- [ ] **Step 1:** Create the scheduled task with `create_trigger`.
  - Daily at 06:4x Africa/Casablanca. Add jitter per the tool guidance and keep it outside 06:00–09:00 Pacific; Casablanca morning qualifies.
  - The prompt: clone `kubicaconsulting/consulting-lead-pipeline` and follow `RUNBOOK.md` for market `es`, never sending email.
- [ ] **Step 2:** Confirm the task's permission mode. Tell Mossaab how to enable automatic approval if runs would otherwise stop for approval.
- [ ] **Step 3:** Commit README update: `docs: schedule and operations`.
