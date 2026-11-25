# Assess one company

Input: a company name, its domain if known, the market profile, and any matching job ads from `passed.json`.
Output: exactly one JSON object matching `model.Assessment`, appended to `run/assessments.json`.

Research with WebSearch and WebFetch: the company site (about, careers, engineering blog), LinkedIn company page snippets, funding news, and the national company register where one is searchable. Spend at most about 6 lookups per company.

Fill every field. Use `unknown` (or `0` for headcount) whenever the evidence is missing. Never guess a value to make a company score higher.

| Field | Allowed values | Evidence rule |
|---|---|---|
| `company`, `domain`, `website` | text | domain without `www.` or path |
| `market` | the run's market code | |
| `industry_tier` | `fintech`, `insurtech`, `travel`, `mobility`, `energy`, `healthtech`, `other_product`, `agency_outsourcing`, `unknown` | Choose by what the company sells. Consultancies, staffing and body shops are `agency_outsourcing` even when the client is a fintech. |
| `headcount` | integer, 0 = unknown | Use a stated figure, or the lower bound of the LinkedIn size band. |
| `funding` | `seed`, `a`, `b`, `c_plus`, `public`, `group`, `bootstrapped`, `unknown` | `group` means part of an established corporate group. |
| `platform` | `scale`, `modernise`, `both`, `product`, `maintenance`, `unknown` | `scale`: building platforms at scale (high traffic, distributed systems). `modernise`: replacing or rebuilding legacy systems, migrations, monolith → services. `maintenance`: the role or team mainly keeps existing systems running. |
| `sponsorship` | `offered`, `prior`, `unknown`, `none` | `offered`: an ad or careers page states visa or relocation support. `prior`: public evidence of hiring non-EU engineers into the market. `none`: explicitly refuses. |
| `local_entity` | `yes`, `no`, `unknown` | Is there a legal entity registered in the market (in Spain, an S.L. or S.A., or an office listed with a CIF)? It is required for the Highly Qualified Professional permit. If the company only hires through an Employer of Record, the answer is `no`. |
| `open_role_urls` | URLs | Matching ads from this run, plus any matching backend role on its careers page |
| `reason` | one sentence | What makes this company a fit or not, citing the strongest fact |
| `sources` | URLs, space-separated | Every page used |

Do not record anything about individual people here.
