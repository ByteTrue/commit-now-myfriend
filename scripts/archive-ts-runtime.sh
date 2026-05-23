#!/usr/bin/env bash
set -euo pipefail

branch_name="${1:-archive/typescript-runtime}"

git branch "$branch_name" HEAD
printf 'Created archive branch: %s\n' "$branch_name"
