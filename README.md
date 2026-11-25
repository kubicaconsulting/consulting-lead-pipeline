# consulting-lead-pipeline

A daily pipeline that finds and qualifies companies and backend roles that match my career goals: senior or lead roles, building platforms at scale or replacing legacy ones, in growing industries (fintech first). It looks at established companies that can sponsor a work permit, and it prepares direct outreach. Spain is the default market. Any market can be added as a profile file.

Nothing is ever sent automatically. Outreach stops at Gmail drafts, which I review and send myself.

## How it works

```
 job boards ──┐                         ┌── Google Sheet "Lead Tracker"
 (Indeed,     │   leadpipe (Go)         │     Companies · Contacts · Leads
 LinkedIn,    ├─► normalize → gate ─┐   │     Dashboard · Run log
 Tecnoempleo) │                     │   │
 discovery ───┘                     ▼   │
 (web search)          Claude: research & assess ──► leadpipe score/rows ──┤
                                    │                                      │
                                    └─► contacts ─► Gmail drafts (never sent)
```

- **`leadpipe`** (this repo, Go) does every deterministic step:
  - normalising raw board data into one job format;
  - extracting salary, seniority, city and work mode, work-permit wording, stack and platform signals;
  - hard gates and dedupe;
  - scores;
  - rows for the sheet.
- **A daily Claude scheduled task** follows [`RUNBOOK.md`](RUNBOOK.md):
  - pulls ads through the Indeed and Apify connectors;
  - runs `leadpipe`;
  - researches each company with [`prompts/assess_company.md`](prompts/assess_company.md);
  - finds hiring contacts with [`prompts/find_contacts.md`](prompts/find_contacts.md);
  - writes drafts with [`prompts/draft_email.md`](prompts/draft_email.md);
  - updates the tracker.

## Qualification rules (Spain profile)

**Hard gates.** A job that fails any of these is excluded, with the reason logged:
- stated salary below €56k gross;
- Barcelona roles can be onsite, hybrid or remote; any other city must be remote;
- excluded if the ad requires existing EU or Spanish work rights ("based in Spain" alone is only flagged);
- excluded if the title is junior or mid level, or if the ad's stated minimum experience is below 5 years (roles asking for 10+ years are flagged and rank higher);
- excluded if the ad doesn't use Python, Go or Node.js/TypeScript (only these three stacks are tracked; others are ignored);
- excluded if the company was assessed in the last 60 days, unless it has posted a new matching role.

**Company score (decides outreach).** Scored out of 100:

| Factor | Weight |
|---|---|
| Industry (fintech first) | 33.3 |
| Maturity (≥200 people, post-Series B, or part of a group) | 26.7 |
| Platform work (scale or modernisation) | 33.3 |
| Sponsorship signals | 6.7 |

- Outreach at **≥ 65**, and only if the company has a legal entity in the market, which the HQP permit requires, and has not stated that it refuses visa sponsorship.

**Role score (ranks ads).** Level and scope 40, salary against the €68k target 30, platform signals in the ad 30.

All values live in [`config/`](config/).

## Usage

```bash
go build -mod=vendor -o leadpipe ./cmd/leadpipe

./leadpipe normalize --kind linkedin --in raw.json --out jobs.json      # appends
./leadpipe gate  --in jobs.json --seen seen.json --out-pass passed.json --out-excluded excluded.json
./leadpipe score --in assessments.json --out scored.json
./leadpipe rows  --kind companies --in scored.json --out company_rows.json
./leadpipe rows  --kind leads --in passed.json --out lead_rows.json
```

Every command takes `--market <code>` (default `es`) and `--config <dir>` (default `config`).

```bash
go test ./...
```

Dependencies are vendored (`vendor/`), so the build works without network access.

## Changing the search

| To change | Edit |
|---|---|
| Salary floor/target, cities, work-permit wording, sources, discovery queries | `config/markets/es.yaml` |
| Weights, pass threshold, industry tiers, re-assessment window, daily target | `config/global.yaml` |
| Add a market (e.g. Portugal) | Copy `config/markets/es.yaml` to `pt.yaml`, adjust it, and run with `--market pt` (or change the scheduled task's prompt) |
| Free → paid data | Set `tier: paid` in `config/global.yaml`. This enables the sources marked `tier: paid` |

## Resources

| What | Where |
|---|---|
| Drive folder | [Spain Job Search](https://drive.google.com/drive/folders/1UtH9y7wEPC4piRhilc8nVStpdU8lBnQC) (`1UtH9y7wEPC4piRhilc8nVStpdU8lBnQC`) |
| Tracker | [Lead Tracker](https://docs.google.com/spreadsheets/d/1u8kJ1UjhqDVkLYUyGKIjW30dwgyN2xIHG73-n3CwltA/edit) (`1u8kJ1UjhqDVkLYUyGKIjW30dwgyN2xIHG73-n3CwltA`). Columns: [`docs/sheet-schema.md`](docs/sheet-schema.md) |
| CV: Senior | [PDF](https://drive.google.com/file/d/1N2wIuHBjSj_CWGlKLlPzavGfWmdWdgV5/view) (`1N2wIuHBjSj_CWGlKLlPzavGfWmdWdgV5`) |
| CV: Lead | [PDF](https://drive.google.com/file/d/187A0pnK8mw_G0NvjERXyR9PylAD7BLe3/view) (`187A0pnK8mw_G0NvjERXyR9PylAD7BLe3`) |
| CV: Architecture | [PDF](https://drive.google.com/file/d/1-ZMg0Smp1eXFdHyuBD1MPP6vwdoUkgQ5/view) (`1-ZMg0Smp1eXFdHyuBD1MPP6vwdoUkgQ5`) |
| Design | [`docs/specs/`](docs/specs/) |
| Build plan | [`docs/superpowers/plans/`](docs/superpowers/plans/) |

The CV PDFs are private in Drive. Before the first drafts go out, set each one to "Anyone with the link can view" so that recipients can open it.

## Schedule

- **When it runs:** a Claude scheduled task, "Daily lead pipeline (Spain)", every day at 06:41 Africa/Casablanca.
- **What each run does:** it clones the default branch, follows `RUNBOOK.md` with `MARKET=es`, and ends with a summary in its session. You get a push notification when a run finishes.
- **Pausing or changing it:** turn the task off in its settings, or edit its prompt. Changing the market in the prompt runs a different profile.
- **Before the first useful run:**
  - merge this branch into `main`;
  - reconnect Gmail with permission to create drafts;
  - set the three CV PDFs to "Anyone with the link can view".

## Privacy

Contacts hold only work-context data: name, title, LinkedIn URL, work email and where it came from. That data is used for one-to-one outreach. Mark `opt_out = yes` on a contact and the pipeline will never draft to them again.
