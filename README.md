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
4. Print a table of out of date actions if any were found
5. Exit with an informational status code:

| code | meaning                                           |
| ---- | ------------------------------------------------- |
| 0    | all actions are up to date                        |
| 1    | some actions are out of date                      |
| 2    | operational error: github outage, invalid command |

### Options

```
aver help     Print help message
aver version  Print version
```

## What counts as "up to date"?

Aver respects the precision of your version specifier:

| You specify | Aver reports outdated if                                     |
| ----------- | ------------------------------------------------------------ |
| `v6`        | A newer major version exists (e.g., `v7`)                    |
| `v6.1`      | A newer minor or major version exists (e.g., `v6.2` or `v7`) |
| `v6.1.0`    | Any newer version exists (e.g., `v6.1.1`, `v6.2.0`, or `v7`) |

For example:

- `actions/checkout@v6` is up to date even if `v6.0.2` exists (you asked for v6, you have v6)
- `actions/checkout@v6.0` would be outdated if `v6.1` exists
- `actions/checkout@v6.0.0` would be outdated if `v6.0.1` exists

SHA-pinned actions (e.g., `@a1b2c3d`) report how many commits behind the default branch they are, unless `--ignore-sha` is passed.

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
