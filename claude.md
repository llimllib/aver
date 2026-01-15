# Aver

GitHub Actions version checker CLI tool.

## Build

```bash
make        # Build binary
make test   # Run tests
make lint   # Run golangci-lint
make clean  # Remove binary
```

Uses Go 1.25: `/Users/llimllib/.local/share/mise/installs/go/1.25.5/bin/go`

## Before Committing

Always run tests and lint before committing:

```bash
make test && make lint
```

Don't commit with any lint errors

Run `aver` to check our own GitHub Actions are up to date:

```bash
aver
```

## Project Structure

```
cmd/aver/main.go     # CLI entry point, flag handling, output formatting
pkg/actions/         # Core logic
  actions.go         # Action discovery, version checking, GitHub API
```

## Key Concepts

- **Action discovery**: Walks `.github/workflows/*.{yml,yaml}`, recursively extracts `uses:` fields
- **Version checking**: Fetches tags from GitHub API, compares semver
- **SHA-pinned actions**: Compares against default branch HEAD, reports commits behind
- **Subdirectory actions**: `actions/cache/restore` extracts repo as `actions/cache`

## Code Style

- No third-party CLI libraries; flags parsed manually in `hasFlag()`
- Supports `--flag`, `-flag`, and `flag` variants (no single-dash requirement)
- Errors for inaccessible repos become warnings, don't fail the whole run
- JSON output via `--json` for scripting

## GitHub API

- Uses unauthenticated requests by default (60/hour rate limit)
- Set `GITHUB_TOKEN` env var for higher limits
- Endpoints used:
  - `GET /repos/{owner}/{repo}/tags` - version tags
  - `GET /repos/{owner}/{repo}` - default branch
  - `GET /repos/{owner}/{repo}/git/ref/heads/{branch}` - branch HEAD
  - `GET /repos/{owner}/{repo}/compare/{base}...{head}` - commit comparison

## Skill

The `skill/github-actions-version-check/` directory contains a skill for AI coding agents. When adding new features or flags to aver, update the skill's SKILL.md to document them.

## Commits

Use conventional commit style: `feat:`, `fix:`, `docs:`, `refactor:`, etc.
