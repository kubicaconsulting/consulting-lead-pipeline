# Find contacts at one passing company

Input: one scored company with `outreach == yes`, the Contacts tab, and `max_contacts` from config.
Output: up to `max_contacts` new rows for the Contacts tab.

Who to look for:

| Company size | Roles, in order of preference |
|---|---|
| Under 200 people | CTO, VP Engineering, Head of Engineering |
| 200 or more | Engineering Manager or Head of Platform/Backend for the team that matches the open role, then a technical recruiter or talent partner |

How:
- Use public professional sources only: the company's team page, engineering blog author pages, conference talk pages, and LinkedIn profile titles in search snippets.
- **Work email:**
  - Use it with `email_status = found` only when it appears publicly in a professional context.
  - Otherwise build it from the company's email pattern (seen on its site, its press contact, or a public commit), and set `email_status = unverified`.
  - If no pattern can be found, leave the email empty and keep the LinkedIn URL.

Rules:
- Store only: name, title, LinkedIn URL, work email, email_status and source. No personal email addresses, phone numbers, home locations or anything about the person's private life.
- Skip anyone already in Contacts for this company, and anyone with `opt_out = yes`.
- People who have left the company (per their own profile) are skipped.
- `outreach_status` starts as `new`. It becomes `drafted` once a draft exists.
