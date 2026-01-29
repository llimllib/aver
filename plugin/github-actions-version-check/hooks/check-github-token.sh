#!/bin/bash

# Read JSON input from stdin
input=$(cat)

# Extract the command from tool_input
command=$(echo "$input" | jq -r '.tool_input.command // ""')

# Check if this is an aver command
if [[ "$command" != *"aver"* ]]; then
  # Not an aver command, allow it
  echo '{}'
  exit 0
fi

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
  # Output JSON to deny the tool use
  cat << 'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "GITHUB_TOKEN environment variable is not set.\n\nThe github-actions-version-check plugin requires GITHUB_TOKEN for reliable operation.\nWithout it, aver uses unauthenticated GitHub API requests limited to 60/hour,\nwhich may cause rate limiting errors.\n\nTo fix this:\n  1. Create a GitHub Personal Access Token at https://github.com/settings/tokens\n  2. Export it: export GITHUB_TOKEN=ghp_xxxxx"
  }
}
EOF
  exit 0
fi

# Token is set, allow the command
echo '{}'
exit 0
