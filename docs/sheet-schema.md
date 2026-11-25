# Lead Tracker: sheet schema

The tracker is a native Google Sheet ("Lead Tracker") in the Drive folder "Spain Job Search". The pipeline appends rows. Columns marked *manual* are yours: the pipeline never overwrites them.

## Companies

`leadpipe rows --kind companies` produces these columns, in this order:

| Column | Meaning |
|---|---|
| market | Market profile code (`es`) |
| company | Company name |
| domain | Website domain; the dedupe key when present |
| website | Homepage URL |
| industry_tier | `fintech`, `insurtech`, `travel`, `mobility`, `energy`, `healthtech`, `other_product`, `agency_outsourcing`, `unknown` |
| headcount | Number of employees (0 = unknown) |
| funding | `seed`, `a`, `b`, `c_plus`, `public`, `group`, `bootstrapped`, `unknown` |
| platform | `scale`, `modernise`, `both`, `product`, `maintenance`, `unknown` |
| sponsorship | `offered`, `prior`, `unknown`, `none` |
| local_entity | `yes`, `no`, `unknown`: does it have a legal entity in the market (needed for the HQP permit) |
| company_score | 0–100 |
| outreach | `yes` when score ≥ pass threshold and local entity is not `no` |
| flags | Comma-separated flags |
| reason | One line explaining the score |
| sources | Links used as evidence |
| open_roles | Matching job URLs, one per line |
| last_assessed | YYYY-MM-DD |
| status | *manual*. Starts as `new`; then `watching`, `applied`, `interviewing`, `rejected`, `closed` |
| notes | *manual* |

## Leads

`leadpipe rows --kind leads` produces: market, posted, company, title, city, mode, salary_lo, salary_hi, main_stack, level, role_score, flags, url, source, status (*manual*, starts `new`), notes (*manual*).

## Contacts

Written by the runner (no Go builder): market, company, name, title, linkedin_url, email, email_status (`found` | `unverified`), source, outreach_status, draft_link, cv_variant, drafted_on, opt_out (*manual*), notes (*manual*).

- `outreach_status` is set by the pipeline to `drafted`, then `contacted` (a message to that address appears in Gmail Sent) and `replied` (a message from that address appears in the Inbox).
- After that you set it yourself to `interviewing` or `closed`.
- The pipeline never drafts to a contact whose `opt_out` is `yes`, and never drafts to the same contact twice.

## Config

A read-only copy of the active settings, written at setup: market, tier, salary_floor, salary_target, pass_threshold, reassess_days, daily_target. Change settings in the repo's `config/`, not here.

## Run log

date, market, candidates, gated_pass, gated_excluded, assessed, passed_threshold, contacts_found, drafts, errors, apify_spend_usd, notes.

## Dashboard

Set up once (Google Sheets syntax):

| Cell | Content |
|---|---|
| A1:A7 | Labels: Companies assessed, Passing (outreach = yes), Average company score, Leads collected, Drafts waiting, Contacted, Replied |
| B1 | `=COUNTA(Companies!B2:B)` |
| B2 | `=COUNTIF(Companies!L2:L,"yes")` |
| B3 | `=IFERROR(ROUND(AVERAGE(Companies!K2:K),1),0)` |
| B4 | `=COUNTA(Leads!C2:C)` |
| B5 | `=COUNTIF(Contacts!I2:I,"drafted")` |
| B6 | `=COUNTIF(Contacts!I2:I,"contacted")` |
| B7 | `=COUNTIF(Contacts!I2:I,"replied")` |
| D1 / D2 | "By status" / `=IFERROR(QUERY(Companies!A2:S,"select R, count(B) where B is not null group by R label R 'status', count(B) 'companies'",0),"no companies yet")` |
| G1 / G2 | "Top companies, last 7 days" / `=IFERROR(SORT(FILTER({Companies!B2:B,Companies!K2:K,Companies!N2:N},Companies!Q2:Q>=TEXT(TODAY()-7,"yyyy-mm-dd"),Companies!B2:B<>""),2,FALSE),"none yet")` |
| G24 / G25 | "Recent runs" / `=IFERROR(SORT(FILTER('Run log'!A2:L,'Run log'!A2:A<>""),1,FALSE),"no runs yet")` |
