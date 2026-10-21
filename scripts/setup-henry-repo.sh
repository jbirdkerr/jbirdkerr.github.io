#!/usr/bin/env bash
# ==============================================================================
# setup-henry-repo.sh
#
# Wrapper around setup-repo.sh configured with defaults for the Henry project:
# - Repo: henry
# - Custom Domain: henry.jbirdkerr.net
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_NAME="${1:-henry}"
CUSTOM_DOMAIN="${2:-henry.jbirdkerr.net}"

exec "$SCRIPT_DIR/setup-repo.sh" "$REPO_NAME" "$CUSTOM_DOMAIN" "${@:3}"
