# Contributing Workflow

## Required Flow

1. Create/confirm an issue before code work.
2. Create a branch from `main` mapped to that issue.
3. Implement and test changes on the branch.
4. Open a PR linked to the issue.
5. Review and merge PR.

## Branch Naming

- Feature: `feat/<issue-number>-<short-slug>`
- Bugfix: `fix/<issue-number>-<short-slug>`
- Chore: `chore/<issue-number>-<short-slug>`

Examples:

- `feat/12-message-storage`
- `fix/23-forwarding-retry`

## Commit Style

Prefer concise commits:

- `feat: add telegram forwarding service`
- `fix: handle flood wait in forwarder`
- `chore: update project templates`

## Pull Request Rules

- PR must reference an issue (`Closes #<id>`).
- PR should focus on one issue.
- Include testing notes.
- Keep secrets out of repository.

## Merge Strategy

Recommended: **Squash and merge** to keep main history clean.

## Labels

Recommended labels:

- `type:feature`
- `type:bug`
- `type:chore`
- `priority:high`
- `priority:medium`
- `priority:low`

## Suggested Initial Backlog

- Forwarding reliability and retry policy
- Message persistence (DB)
- API for stored messages
- Auth for dashboard/API
- Observability (metrics/log enrichment)
