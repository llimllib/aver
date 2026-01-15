---
name: github-actions-version-check
description: Check and update GitHub Actions versions. Use when creating, modifying, or reviewing GitHub Actions workflow files (.github/workflows/*.yml), or when asked to ensure actions are up to date.
---

# GitHub Actions Version Check

Ensures GitHub Actions in workflow files use up-to-date versions.

## Prerequisites

Install aver (GitHub Actions version checker):

```bash
go install github.com/llimllib/aver/cmd/aver@latest
```

Or if aver is already in PATH, skip this step.

## Workflow

### When Creating or Modifying Workflow Files

1. **Before adding any action**, check what the latest version is:
   ```bash
   # Search GitHub for the action's latest release
   # Example: for actions/checkout, check https://github.com/actions/checkout/releases
   ```

2. **Always use the latest major version** when adding new actions:
   - Good: `actions/checkout@v4`
   - Avoid: `actions/checkout@v3` (unless v3 is actually latest)

3. **After making changes**, validate with aver:
   ```bash
   aver
   ```

4. **If outdated actions are found**, update them to the latest versions shown.

### Checking Existing Workflows

Run aver in the project root:

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

### Understanding Output

```
File                        Action            Current  Latest
--------------------------  ----------------  -------  ------
.github/workflows/ci.yml    actions/checkout  v3       v4
```

This means `actions/checkout@v3` should be updated to `actions/checkout@v4`.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All actions up to date |
| 1 | Outdated actions found |
| 2 | Error occurred |

## Best Practices

1. **Pin to major versions** (`@v4`) not full semver (`@v4.1.2`) unless you need reproducibility
2. **Run aver before committing** workflow changes
3. **For SHA-pinned actions**, aver shows commits behind - update periodically
4. **Set GITHUB_TOKEN** for higher API rate limits:
   ```bash
   export GITHUB_TOKEN=ghp_xxxxx
   ```

## Common Actions and Their Repos

| Action | Repository |
|--------|------------|
| `actions/checkout` | https://github.com/actions/checkout |
| `actions/setup-node` | https://github.com/actions/setup-node |
| `actions/setup-python` | https://github.com/actions/setup-python |
| `actions/setup-go` | https://github.com/actions/setup-go |
| `actions/cache` | https://github.com/actions/cache |
| `actions/upload-artifact` | https://github.com/actions/upload-artifact |
| `actions/download-artifact` | https://github.com/actions/download-artifact |
