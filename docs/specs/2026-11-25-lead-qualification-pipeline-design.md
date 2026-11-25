# Lead & Company Qualification Pipeline — Design

Date: 2026-11-25 · Owner: Mossaab · Status: approved 2026-11-25

## 1. Intent

Mossaab (senior backend engineer: Python/FastAPI, Go/Gin, Node.js; AWS incl. EKS/ECR/EC2/S3/Kinesis; Terraform at working level) is relocating from Morocco to Spain via the **Highly Qualified Professional (HQP) permit**, which requires an employer to sponsor him. He wants a daily automation that finds and qualifies companies and roles aligned with his goals — leadership, building scalable platforms, industry leadership — and prepares direct outreach, so his effort goes to the few targets worth it.

Two lead sources, one database:
- **Job ads** (reactive): new ads on LinkedIn, Tecnoempleo, InfoJobs, Indeed.
- **Company discovery** (proactive): suitable companies with or without open ads, to approach directly via named contacts.

## 2. Goals, non-goals, anti-goals

**Goals**
- Run daily; assess **≥10 new companies per day** (assessed, not necessarily passing).
- Score companies and roles against the criteria in §4 and explain every score in one line.
- Maintain a company/contact/lead database he can edit.
- For passing companies, find 1–3 hiring-relevant contacts and create **Gmail drafts** for him to review.
- Run on free tiers first; switching to paid tiers is a config change, not a rebuild.

**Non-goals**
- Sending any email. Drafts only; Mossaab validates and sends manually. (Hard rule.)
- LinkedIn messaging automation, applying to jobs, CV generation.
- Markets outside Spain.

**Anti-goals (failure even if it "works")**
- Top of the list filled with roles that cannot sponsor a non-EU hire.
- Generic drafts that read as mass mail.
- Storing personal (non-work) contact data, or contacting the same person twice.
- Re-assessing the same companies daily to hit the count.

## 3. Constraints

- **Budget: free tier first.** Apify free $5/month (≈1,400 ads/month ≈ 45/day based on the Nov benchmark cost). Current Apify cycle is exhausted until reset → until then ads come from the Indeed connector only. No paid email-finder.
- **Runtime:** a Claude scheduled task (daily, fresh session each run). Workspace shell cannot reach `api.apify.com`; Apify is reached via the Apify MCP connector only. Web research via WebSearch/WebFetch. Connected: Apify, Indeed, Gmail, Google Drive, Google Sheets.
- **GDPR:** contacts are EU professionals. Store only work-context data (name, title, LinkedIn URL, work email, source). One-to-one outreach only; honour opt-outs.
- **HQP permit:** salary threshold €41,356.36 gross/yr (2026) — below his own floor, so not a separate filter.

## 4. Qualification criteria (as decided with Mossaab)

### 4.1 Hard gates (fail → excluded, reason logged)
| Gate | Rule |
|---|---|
| Sponsorship | Exclude if the ad/company explicitly requires existing EU work rights / no sponsorship. Silent → pass, flagged `sponsorship: unknown`. |
| Salary | If stated: exclude below **€56k gross**. Not stated → pass, flagged `salary: unknown`. |
| Location | Barcelona: office, hybrid or remote all OK. Madrid / any other Spanish city: **fully remote only**. |
| Seniority | Exclude junior/mid titles. |
| Stack | Backend role with Python, Go or Node.js/TypeScript in the main stack (benchmark `classify_stack` logic). Company-only leads: has backend engineering in these stacks. |

### 4.2 Scores (0–100 composite; weights are builder's starting point, tunable in config)
| Dimension | Weight | What scores high |
|---|---|---|
| Industry | 25 | Fintech first; then other growing industries (insurtech, travel, mobility, energy, healthtech). Agencies/outsourcing/body shops low. |
| Company maturity | 20 | ~200+ headcount, or post-Series B, or part of an established group. <50 people heavily penalised (not excluded). |
| Platform work | 25 | Building platforms at scale (platform, high traffic, distributed systems) or modernising legacy (legacy, migration, re-architecture, monolith→microservices, replatforming). Maintenance-only low. |
| Role level & scope | 15 | Senior and Lead/Staff/Principal equal top; Engineering Manager small penalty. Senior-responsibility signals (system design, architecture ownership, mentoring) rank within band. |
| Salary | 10 | ≥€68k gross (≈ €48k net) full; €56–68k partial; unknown neutral-low. |
| Sponsorship signal | 5 | Explicit relocation/visa support > unknown. Known prior sponsorship of non-EU hires counts. |

Pass threshold for outreach: builder's call, start at 65 and tune after week 1.

### 4.3 Profile reference
Fit/gap logic reuses `extract.py` REQ list (statuses as confirmed 25 Nov: FastAPI/Gin/AWS EKS-ECR-EC2-S3 = Have; Terraform, Kubernetes = Partial).

## 5. Approaches considered

- **A. Claude scheduled task, rules + judgement** — deterministic gates in Go, Claude judges industry/maturity/platform fit and researches companies. Handles fuzzy criteria; non-deterministic and uses session time.
- **B. Self-hosted cron script (GitHub Actions), rules only** — cheap and predictable, but keyword rules misjudge maturity/industry and can't research companies.
- **C. Curated company watchlist** — highest quality, misses unknown companies.

**Chosen: A + C.** Ads and company discovery both feed one queue; companies scoring well enter a watchlist whose careers pages are rechecked. Chosen because the core criteria need judgement and research, and Mossaab wants proactive outreach, not only ads.

## 6. Design

### 6.1 Daily run (one scheduled task, early morning Africa/Casablanca)
1. **Load state** from the Google Sheet (companies, contacts, leads, config).
2. **Collect candidates**
   - Ads: Indeed connector (always); Apify LinkedIn/Tecnoempleo/InfoJobs actors when budget allows (daily ad cap in config). Only ads posted since last run.
   - Discovery: web search for Spanish fintech / growing-industry companies (funding news, scale-up directories, Barcelona tech ecosystem lists), plus watchlist careers pages.
3. **Dedupe**: skip companies assessed in the last **60 days** unless a new matching ad appeared.
4. **Gate** (Go, §4.1) → excluded with reason.
5. **Assess ≥10 companies** (Claude + web research): fill company fields, score §4.2, one-line reason. If fewer than 10 new candidates, top up from discovery.
6. **Contacts** for companies ≥ pass threshold: 1–3 people (CTO/VP Eng at smaller companies; Eng Manager/Head of Platform + tech recruiter at larger). Work email from public sources, else inferred from the company's email pattern and marked `unverified`.
7. **Drafts**: one Gmail draft per new contact, in English, citing something specific about the company's platform/legacy work and pointing to the matching CV variant (Senior / Lead / Architecture). Never sent.
8. **Write back** to the Sheet and post a short daily summary (assessed count, passes with reasons, links to new drafts).

### 6.1b Where things live (decided by Mossaab: code in private GitHub, dashboard in Google Drive)
- **Private GitHub repo** (under `kubicaconsulting`, created by Mossaab, attached to the session): **Go** module (decided by Mossaab) for collection/normalising/gates/dedupe, porting the extraction rules from the benchmark's `normalize.py`, `filter.py`, `extract.py`, `config.yaml` (weights, thresholds, cities, salary floors, tier), prompt files for company assessment, contact search and draft writing, and `RUNBOOK.md`, the step-by-step instructions each scheduled run follows. No secrets in the repo; auth stays in the connectors (Apify, Indeed, Gmail, Drive, Sheets).
- **Runner:** a daily Claude scheduled task. Each run clones the repo, runs the deterministic steps with the Go binary, does the judgement steps per `RUNBOOK.md`, and writes results through the connectors. Changes to the pipeline go through commits/PRs, so every run uses the latest reviewed version.
- **Google Drive folder "Spain Job Search"**: the tracker as a **native Google Sheet** (not an .xlsx file), so the pipeline can update cells in place without overwriting Mossaab's edits; it exports to Excel at any time. Plus a `Dashboard` tab (counts by status, score distribution, top companies this week, run log) and a link to the CV v3 folder.
- Later option: move the deterministic part to GitHub Actions if run cost/time becomes an issue; judgement steps stay with Claude.

### 6.2 Database (Google Sheet in Mossaab's Drive)
- **Companies**: name, website, industry, HQ/Spain offices, headcount, funding stage, platform/legacy notes, sponsorship evidence, open matching roles, score, score breakdown, reason, status, last_assessed, source.
- **Contacts**: company, name, title, LinkedIn URL, work email, email_status (found/unverified), source, outreach status (not contacted / drafted / contacted / replied / interviewing / closed), draft link, opt-out.
- **Leads (ads)**: company, title, URL, board, posted, city, remote, salary, stack, gate result, score, reason.
- **Config**: weights, thresholds, daily targets, ad cap, tier (free/paid), excluded companies.
- **Run log**: date, candidates, assessed, passed, drafts, errors, Apify spend.

Mossaab may edit any status/notes cell; the pipeline must not overwrite his manual edits (status, notes, opt-out).

### 6.3 Free → paid switch
`tier` in Config. Paid: raise ad cap, enable Apify company/contact actors, add email verification service. No structural change.

### 6.4 Markets are configurable (decided by Mossaab)
Spain is the default market, but every market-specific rule lives in a **market profile** in config, selected per run (`--market es` by default):
- country, currency, accepted cities and work modes (Spain: Barcelona any mode; other cities remote only);
- salary floor and target in local currency and gross (Spain: floor €56k, target €68k);
- immigration route and its gates (Spain: HQP permit, local legal entity required, EU-work-rights wording excluded), including the local-language wording to match;
- job boards and Apify actors to query, and the Indeed country code;
- discovery queries and sources (e.g. Spanish fintech directories) and ad/outreach language.
Adding a market means adding a profile file, not changing code. Industry priority, seniority rules and score weights are global by default and can be overridden per market. The database keeps a `market` column so several markets can share one tracker.

## 7. Decided by Mossaab
Pipeline over one-off tool; daily; ≥10 companies/day; ads + direct company outreach; criteria in §4 (fintech first, mature companies, platform building or legacy rebuild, senior with leadership open, €56k gross floor / €48k net target, Barcelona any mode, elsewhere remote only); contact database with names/titles/emails; stop at Gmail drafts, never send; start free, upgradeable; code in a private GitHub repo; tracker/dashboard in Google Drive; market configurable with Spain as default; Go for the code; CV v3 (Senior / Lead / Architecture) stored in Drive › CV › v3 (Nov 2026), and drafts link the matching PDF.

## 8. Builder's calls (proposed by Claude, open to correction)
Weights and pass threshold; 60-day re-assessment window; Google Sheet as database; contact roles by company size; draft language English; run time; discovery sources and queries; how the sponsorship history is evidenced. Plus, settled after review:
- **Two scores, not one.** *Company score* (industry, maturity, platform/legacy work, sponsorship signal, reweighted to 100) decides outreach. *Role score* (level & scope, salary, platform signals in the ad) ranks ads. A company-only lead is never penalised for having no ad.
- **Residence wording.** "Must already have EU/Spanish work rights / NIE" → excluded. "Based in Spain" / "residir en España" alone → pass, flagged, since sponsorship would bring him there.
- **Spanish entity.** HQP needs a Spain-registered employer. Companies with no Spanish entity → excluded from outreach (EOR-only hiring flagged, not counted).
- **What counts toward ≥10/day:** companies that pass the gates and get a full assessment. Gate failures are logged but don't count.
- **Outreach status:** pipeline searches Gmail Sent for each drafted contact's address to mark `contacted`, and Inbox for replies to mark `replied`; later stages and opt-outs are updated manually in the Sheet. A contact is never drafted twice. Drafts to `unverified` addresses are created but flagged in the summary.

## 9. Left to the builder
Script structure, prompts for the scoring step, sheet formatting, error handling and retries, how the run summary is delivered.
