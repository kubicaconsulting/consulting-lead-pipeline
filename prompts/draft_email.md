# Draft one outreach email

Input: one contact row, its company's assessment, any matching open role, and the CV links in `README.md`.
Output: one Gmail draft created with `create_draft`. **Never call a send tool.** Write the draft link to the contact's `draft_link`, set `outreach_status = drafted` and `drafted_on`.

Write in the market profile's `outreach_lang` (English for `es`).

**Subject:** short and specific. Name the role, or the platform or legacy area, e.g. "Senior backend engineer: payments platform rebuild". Avoid generic subjects.

**Body:** under 150 words, plain text, no bullet lists.
1. One sentence on something specific about their platform or legacy work, taken from the assessment's sources. No flattery.
2. Two sentences on the most relevant matching experience, taken only from the CV v3 text. Never invent facts, metrics or titles. The usual anchors:
   - helped decompose a monolith into domain microservices for a platform handling nearly 40 million photos a day at peak;
   - the zero-downtime migration of nearly 1 million accounts;
   - Go, Python and Node.js services on AWS EKS.
3. The ask: a short call about the open role, or about backend roles in their team if none is open. Mention plainly that he is relocating to Spain and would need visa sponsorship through the Highly Qualified Professional permit.
4. A link to the matching CV PDF, then sign off with "Mossaab Kanzit" and his LinkedIn URL.

**Which CV to link:**

| Situation | CV |
|---|---|
| The company's `platform` is `modernise` or `both` | Architecture |
| The contact is CTO or VP Engineering at a company under 200 people | Lead |
| Anything else | Senior |

Style: write like a person. No dashes as connectors, no "not X but Y" contrasts, no lists of three for rhythm, no "I hope this finds you well", no exclamation marks.
