# GitHub Actions Version Check Plugin

A Claude Code plugin that helps ensure GitHub Actions in your workflow files use up-to-date versions.

## Features

- **Skill**: Automatically invoked when working with GitHub Actions workflow files
- **Token Validation**: Blocks `aver` commands if `GITHUB_TOKEN` is not set, with a helpful error message

## Installation

### Prerequisites

1. Install [aver](https://github.com/llimllib/aver) (GitHub Actions version checker):
   ```bash
   go install github.com/llimllib/aver/cmd/aver@latest
   ```

2. Set up your GitHub token for API access:
   ```bash
   export GITHUB_TOKEN=ghp_xxxxx
   ```

### Install the Plugin

#### From the aver Repository

First, add the aver repository as a marketplace:

```shell
/plugin marketplace add llimllib/aver
```

Then install the plugin:

```shell
/plugin install github-actions-version-check@aver
```

Or install directly from the command line:

```bash
claude plugin install github-actions-version-check@aver
```

#### From Local Directory (Development)

```bash
claude --plugin-dir /path/to/plugin/github-actions-version-check
```

## Usage

The plugin automatically activates when you:

- Create or modify GitHub Actions workflow files (`.github/workflows/*.yml`)
- Ask Claude to check if actions are up to date
- Review workflow files for outdated dependencies

### Manual Commands

You can also run aver directly:

```bash
# Human-readable output
aver

# JSON output for scripting
aver --json

# Ignore SHA-pinned actions
aver --ignore-sha

# Only report major version updates
aver --ignore-minor
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | Recommended | GitHub Personal Access Token for API access. Without it, requests are limited to 60/hour and the plugin will block aver commands. |

## Plugin Structure

```
github-actions-version-check/
├── .claude-plugin/
│   └── plugin.json          # Plugin manifest
├── skills/
│   └── github-actions-version-check/
│       └── SKILL.md         # Skill definition
├── hooks/
│   ├── hooks.json           # Hook configuration
│   └── check-github-token.sh # Token validation script
└── README.md
```

## License

MIT
