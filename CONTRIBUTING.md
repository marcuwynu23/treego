# Contributing to TreeGo

Thank you for your interest in improving TreeGo. This document explains how to report issues, propose changes, and submit pull requests.

## Code of conduct

Everyone participating in this project is expected to follow the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). Unacceptable behavior may be reported to **help@marcuwynu.space**.

## How to contribute

- **Bug reports** — Use the [bug report issue template](.github/ISSUE_TEMPLATE/bug_report.md) and include steps to reproduce, expected vs actual behavior, and your environment (OS, Go version).
- **Feature ideas** — Open an issue using the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) so we can discuss scope and design before you invest a lot of time.
- **Pull requests** — Prefer small, focused PRs with a clear description. Fill out the [pull request template](.github/pull_request_template.md) when you open a PR.

## Development setup

1. Install [Go](https://go.dev/dl/) (this repo targets **Go 1.23.5**; see `go.mod`).
2. Clone the repository and work from the project root.
3. Use the Makefile for common tasks (on Windows, install **Make** if your shell does not provide it — the CI workflow does this for tests).

```bash
make tidy
make fmt
make vet
make test
```

For race-detector checks (closer to CI):

```bash
make test-race
```

## Guidelines

- Match existing style: formatting (`go fmt`), naming, and structure.
- Add or update tests when behavior changes; keep tests deterministic.
- Update user-facing docs (`README.md`, flag help text) when flags or behavior change.
- Keep commit messages clear; conventional prefixes (`feat:`, `fix:`, `docs:`, `chore:`) are welcome.

## Questions

If something is unclear, open an issue with the **question** label (or describe it in a feature request) and we will help from there.
