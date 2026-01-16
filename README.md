# aver

A GitHub **A**ctions **ver**sion checker. Scans your GitHub actions workflow files and reports outdated versions.

## Installation

### Homebrew (recommended)

```bash
brew install llimllib/tap/aver
```

### Alternatives

`go install`

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
Outdated actions:
File                        Action            Current  Latest
--------------------------  ----------------  -------  ------
.github/workflows/lint.yml  actions/checkout  v5       v6.0.2
.github/workflows/lint.yml  actions/setup-go  v5       v6.2.0

SHA-pinned actions behind default branch:
File                        Action            Current SHA  Latest SHA  Branch  Behind
--------------------------  ----------------  -----------  ----------  ------  ------
.github/workflows/lint.yml  actions/checkout  a1b2c3d      e5f6g7h     main    12
```

In terminals that support [OSC 8 hyperlinks](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda), action names link to their GitHub repository, version numbers link to their release tags, and SHA values link to their commits.

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

SHA-pinned actions (e.g., `@a1b2c3d`) report how many commits behind the default branch they are, along with the latest SHA on that branch, unless `--ignore-sha` is passed.

## Using with AI Coding Agents

Aver works well with AI coding agents like [Claude Code](https://claude.ai/code) and [Pi](https://github.com/badlogic/pi-coding-agent) to prevent them from adding outdated GitHub Actions.

### Option 1: Add to CLAUDE.md / AGENTS.md

Add this to your project's `CLAUDE.md` or `AGENTS.md`:

```markdown
## GitHub Actions

When creating or modifying `.github/workflows/` files:

1. Run `aver` to check for outdated actions before committing
2. Always use the latest major version for any new actions
3. If aver reports outdated actions, update them to the versions shown
```

### Option 2: Install the Skill

For [Pi](https://github.com/badlogic/pi-coding-agent) or [Claude Code](https://claude.ai/code), install the included skill for automatic version checking guidance:

```bash
# For Pi
cp -r skill/github-actions-version-check ~/.pi/agent/skills/

# For Claude Code
cp -r skill/github-actions-version-check ~/.claude/skills/
```

The agent will automatically load the skill when working with GitHub Actions workflow files.

**I have not actually used this**, please let me know how it works for you and if there are any changes you'd like to make.

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
make release  # Make a release
```

## License

MIT
