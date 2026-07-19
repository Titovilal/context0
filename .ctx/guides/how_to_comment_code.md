# Code Comment Guidelines

Standard for comments in this repository. Also the rubric for AI review agents that audit PRs and trim comment noise. The bar: **few, short, and load-bearing** — a professional codebase averages far fewer comments than AI-generated code produces.

## Core principle

A comment exists to state something the code **cannot** express: a constraint, a non-obvious "why", a trap for the next editor. If deleting the comment loses no information a competent reader couldn't recover from the code itself, the comment is noise — delete it.

Corollary: prefer making the code self-explanatory (better name, extracted function, clearer structure) over adding a comment that compensates for unclear code.

## What a comment MAY do (keep these)

- **Explain why, not what.** A surprising decision, a rejected alternative, a business rule:
  ```ts
  // Telegram silently drops messages >4096 chars; split before sending
  ```
- **Warn about a trap** that isn't visible at the call site:
  ```ts
  // Must run before the Stripe webhook handler mutates the subscription row
  ```
- **State an invariant or constraint** the types can't encode:
  ```ts
  // items is already sorted by dueAt ascending (see scheduler query)
  ```
- **Link an external cause**: an upstream bug, an RFC, a provider quirk. Include the reference.
- **Mark intentional weirdness** so nobody "fixes" it:
  ```ts
  // Intentionally not awaited: fire-and-forget analytics
  ```
- **TODO/FIXME with substance** — what's missing and why it's deferred, not a vague "improve this".
- **Public API doc comments** (JSDoc on exported functions of shared packages) when the signature alone doesn't convey contract, units, or failure modes. One or two lines; don't restate parameter names.

## What a comment MUST NOT do (delete these)

- **Narrate the next line.** `// increment the counter`, `// loop over users`, `// return the result`.
- **Restate the name.** `// Sends the email` above `sendEmail()`.
- **Section headers inside short functions.** `// Step 1: validate`, `// Step 2: save`. If steps need labeling, extract functions.
- **Talk to the reviewer.** `// Changed this to fix the bug`, `// This is now correct`, `// Updated per feedback`. That belongs in the commit message or PR description, never in the code.
- **Describe the diff or the past.** `// Previously this used X`, `// New implementation`. Git history already records it.
- **Restate types.** `// userId: the id of the user`.
- **Dead code in comment form.** Commented-out blocks — delete; git remembers.
- **Placeholder filler.** `// Helper functions`, `// Constants`, `// Imports`.
- **Explain standard language/library behavior.** `// ?? returns the right side if null`.

## Style rules for surviving comments

- One line when possible; two at most. If it needs a paragraph, the code probably needs restructuring — or the explanation belongs in `.ctx/docs/`.
- Written in **English**, matching the codebase.
- Sentence fragment is fine; no trailing period needed on single-liners. Be consistent with the surrounding file.
- Place the comment on its own line above the code it governs, not trailing at end-of-line (except very short clarifiers).
- Keep comments **adjacent** to what they explain. A comment far from its subject rots fastest.
- When code changes, its comments are part of the diff: update or delete them. A stale comment is worse than none.

## Rubric for AI review agents

When reviewing a PR's comments, apply this test to **each comment** in the diff:

1. **Delete test**: hide the comment; does a competent reader lose real information? If no → **delete**.
2. **Why test**: does it explain what/how instead of why? If a rename or extraction would make it redundant → **prefer the refactor if trivial, otherwise delete the comment**.
3. **Length test**: >2 lines? Compress to the constraint/warning only, or move the essay to the relevant `.ctx/docs/` file.
4. **Reviewer-talk test**: does it reference the change itself ("now", "fixed", "updated", "refactored to")? → **delete**, no exceptions.
5. **Staleness test**: does the comment still match the code in this diff? If the code moved on → update or delete.

Do **not** add new comments while trimming, except: a genuinely dangerous trap you discovered while reading, with a one-line warning. Never add narration to "compensate" for deletions.

Expected outcome on typical AI-generated PRs: **most comments deleted**, a handful kept or tightened. Zero comments in a file is a perfectly healthy state.

## Calibration examples

```ts
// ── DELETE: narrates the code ──
// Get the user from the database
const user = await db.query.users.findFirst({ where: eq(users.id, id) })

// ── DELETE: talks to the reviewer ──
// Refactored to use the shared helper
const total = sumCredits(entries)

// ── KEEP: non-obvious constraint ──
// Stripe sends duplicate webhooks; upsert keyed on event.id makes this idempotent
await recordEvent(event)

// ── KEEP: trap ──
// date-fns startOfDay uses the server TZ — convert to the user's TZ first
const dayStart = startOfDay(toUserTz(now, user.timezone))

// ── TIGHTEN: right idea, too long ──
// Before: 5 lines explaining the whole retry strategy and its history
// After:
// 429s from Gemini spike at :00; jittered backoff avoids thundering herd
```
