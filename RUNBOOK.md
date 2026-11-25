# RUNBOOK: daily run

Each scheduled run follows these steps in order. `MARKET` is the market code given in the task prompt (default `es`). All scratch files go in `run/` (git-ignored). IDs referenced below are in `README.md` → *Resources*.

**Hard rule: never send email.** Only `create_draft`. No step calls a send, forward or reply tool.

## 1. Build

```bash
git clone --depth 1 https://github.com/kubicaconsulting/consulting-lead-pipeline && cd consulting-lead-pipeline
go build -mod=vendor -o leadpipe ./cmd/leadpipe
mkdir -p run
```

Read `config/global.yaml` and `config/markets/$MARKET.yaml`. They hold the thresholds, sources and discovery queries used below.

## 2. Load tracker state

- Read the Companies tab (`get_values`, `Companies!A2:Q`) of the Lead Tracker sheet.
- Write `run/seen.json`: one `SeenCompany` per row of this market, with
  - `key`: the domain (column C) if present, else the company name (column B) passed through the same normalisation as `gate.Key`: lowercase, legal suffixes removed (SL, SA, Inc...), only letters kept;
  - `name`: the company name (column B) exactly as written, so ads without a domain still match;
  - `last_assessed`: column Q, written as `YYYY-MM-DD`; convert it to RFC3339 midnight UTC (`2026-11-25T00:00:00Z`);
  - `role_urls`: column P, split by newline.
- Read the Contacts tab (`Contacts!A2:N`) to know who has already been contacted or has opted out.

## 3. Collect ads

For each source in the profile whose `tier` is allowed by the profile's `tier`:

- **`via: indeed_connector`**:
  - Call the Indeed `search_jobs` tool for each query and location (`country` from `input`), keeping results from the last 2 days.
  - For promising titles, call `get_job_details`.
  - Write the results as a JSON array of `model.Job` objects to `run/indeed.json`:
    - fields: `id`, `source: "indeed"`, `title`, `company`, `location`, `city`, `url`, `desc`, `posted`, `salary_text`;
    - `work_mode` when stated.
- **`via: apify`**:
  - Check the remaining Apify credit first. If a run would exceed it, skip the source and note "apify credit exhausted" in the Run log.
  - Otherwise call the actor with `input` plus each query, then fetch the dataset items into `run/<name>.json`.

Then normalise each file:

```bash
./leadpipe normalize --market $MARKET --kind generic  --in run/indeed.json   --out run/jobs.json
./leadpipe normalize --market $MARKET --kind linkedin --in run/linkedin.json --out run/jobs.json   # if collected
./leadpipe gate --market $MARKET --in run/jobs.json --seen run/seen.json --out-pass run/passed.json --out-excluded run/excluded.json
```

## 4. Pick today's companies

1. **Companies from `run/passed.json`.** Group the jobs by `dedupe_key`; one company may have several jobs.
2. **Top up.** If fewer than `daily_target` companies, run the profile's `discovery_queries` with WebSearch. Add companies that match the target profile (growing industry, fintech first, building or modernising platforms, established) and are not already in `run/seen.json`.
3. **Cap.** Take at most `daily_target` + 2 candidates, so a few that turn out unusable can be dropped.

## 5. Assess

For each candidate, follow `prompts/assess_company.md` and append one `Assessment` object to `run/assessments.json`. Stop once `daily_target` companies have a full assessment.

```bash
./leadpipe score --market $MARKET --in run/assessments.json --out run/scored.json
./leadpipe rows  --market $MARKET --kind companies --in run/scored.json --out run/company_rows.json
./leadpipe rows  --market $MARKET --kind leads     --in run/passed.json --out run/lead_rows.json
```

Append `run/company_rows.json` to `Companies!A1` and `run/lead_rows.json` to `Leads!A1` with `append_values`.

**If a company is already in the sheet** (it posted a new role), update its row in place instead of appending a new one:
- columns A–Q only;
- never touch R (`status`) or S (`notes`).

## 6. Contacts and drafts

For each scored company with `outreach == true`:

1. Follow `prompts/find_contacts.md`. Append the new contacts to the Contacts tab.
2. For each new contact, follow `prompts/draft_email.md`:
   - create the Gmail draft with `create_draft`;
   - write `draft_link`, `cv_variant`, `drafted_on`;
   - set `outreach_status = drafted`.
3. **Never draft for:**
   - a contact with `opt_out = yes`;
   - a contact that already has a `draft_link`;
   - a company whose `local_entity` is `no`.

## 7. Update outreach status

- For every contact with `outreach_status` = `drafted` or `contacted`, search Gmail:
  - Sent, `to:<email> newer_than:60d`: a match means `contacted`;
  - Inbox, `from:<email>`: a match means `replied`.
- Update column I only; leave every other column as it is.

## 8. Log and report

Append one Run log row: date, market, candidates, gated_pass, gated_excluded, assessed, passed_threshold, contacts_found, drafts, errors, apify_spend_usd, notes.

Then end the run with a short summary as your reply in this session (do not email it):
- how many companies were assessed and how many passed;
- each passing company with its score and the one-line reason;
- links to the new drafts, with a reminder that they are waiting for review and nothing was sent;
- any errors, or sources that were skipped.

## Failure handling

- **A connector call fails:** retry it once. If it still fails, skip that step, record it in `errors`, and keep going.
- **`leadpipe` exits non-zero:** stop the run, log the error, and report it. Don't write partial rows.
- **Fewer than `daily_target` companies could be assessed:** say why in the Run log notes (no new ads, discovery exhausted, credit exhausted).
