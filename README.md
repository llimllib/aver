# aver

A GitHub **A**ctions **ver**sion checker. Scans your workflow files and reports outdated actions.

## Installation

```bash
go install aver/cmd/aver@latest
```

Or build from source:

```bash
make
```

## Usage

Run `aver` in any directory within a Git repository:

```bash
$ aver
File                        Action            Current  Latest
--------------------------  ----------------  -------  ------
.github/workflows/lint.yml  actions/checkout  v5       v6.0.2
.github/workflows/lint.yml  actions/setup-go  v5       v6.2.0
```

The tool will:

1. Find the project root (directory containing `.git` or `.github`)
2. Scan all workflow files in `.github/workflows/*.yml` and `.github/workflows/*.yaml`
3. Check each action's version against GitHub's latest major version tag
4. Exit with code 0 if all actions are up to date, or code 1 with a table of outdated actions

### Options

```
aver help     Print help message
aver version  Print version
```

## What counts as "up to date"?

An action is considered up to date if its major version matches the latest major version tag available on GitHub.

For example:

- `actions/checkout@v4` is outdated if `v5` or `v6` tags exist
- `actions/checkout@v6` is up to date if `v6` is the highest major version
- `actions/checkout@v6.1.0` is up to date (major version 6 matches)
- SHA references (e.g., `actions/checkout@a1b2c3d`) are assumed up to date (cannot compare)

## GitHub API Rate Limits

The tool uses the GitHub API to fetch tags. Unauthenticated requests are limited to 60 per hour.

To increase the rate limit, set a GitHub token:

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

## Development

```bash
make          # Build the binary
make test     # Run tests
make lint     # Run golangci-lint
make clean    # Remove binary
```

## License

MIT
