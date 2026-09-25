---
name: gh
description: GitHub workflow for this repo — read before any code change. Creating issues, issue branches, running tests, opening PRs. Use this whenever you want to change code, commit, push or open a pull request.
---

# GitHub workflow

Repo: `praetorianer777/kurswechsel` · default branch: `main`

## Rules

- Every change needs an issue and a branch named `<type>/<issue>-<slug>`.
  Types: `feat fix chore docs refactor test perf ci build revert`.
- Work only through issues. Something found along the way that is out of scope gets its own issue,
  not an unplanned change on the current branch.
- Never push to `main`. Never merge yourself — that is the user's decision.
- Always call `gh` non-interactively: `--title` / `--body-file`, never open the editor.
- `.claude/hooks/branch-guard.sh` enforces this; a violation is blocked, not commented on.

## Language

Everything in the repository is written in English: issues, pull requests, commit messages, code,
comments and documentation. The only exception is what visitors of the website read, which is
German and lives in `frontend/src/i18n/de.ts`.

## Recipe

```bash
gh auth status                       # preflight

gh issue list --limit 20             # does the issue already exist?
gh issue create --title "..." --body-file /tmp/body.md

gh issue develop 42 --name feat/42-timeline-api --base main --checkout

# ... work ...

./run-tests.sh                       # the hook runs this before every push anyway
git push -u origin HEAD

gh pr create --title "..." --body-file /tmp/pr.md   # body contains "Closes #42"
gh pr checks --watch
gh run view --log-failed             # when CI is red
```

While an earlier PR is still open, a follow-up branch may stack on it:
`gh issue develop 43 --name feat/43-x --base feat/42-timeline-api --checkout` and
`gh pr create --base feat/42-timeline-api`. GitHub retargets it to `main` once the base is merged.

## Pitfalls

- `gh issue develop` creates the branch on the remote and checks it out — no manual
  `git switch -c` needed.
- Branch slugs are lowercase ASCII: `feat/42-timeline-api`, not `feat/42-Timeline-API`.
- Commit messages are Conventional Commits, lowercase, imperative, subject ≤ 72 characters, and
  describe the **effect** rather than the files touched.
- The PR number goes at the end of the subject once it is known.
