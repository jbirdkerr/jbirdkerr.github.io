#!/usr/bin/env bash
# ==============================================================================
# setup-repo.sh
#
# Generalized setup script for GitHub Pages repositories:
# - Copies project assets from a source directory to a target directory.
# - Preserves existing README.md or generates a clean fallback.
# - Supports both static branch deployment and GitHub Actions workflow deployment.
# - Sets up Git, creates the GitHub repository with `gh`, pushes, and configures Pages.
# ==============================================================================

set -euo pipefail

show_help() {
  cat <<EOF
Usage: $(basename "$0") [options] [REPO_NAME] [CUSTOM_DOMAIN] [TARGET_DIR] [SOURCE_DIR]

Options:
  -r, --repo <name>         GitHub repository name
  -d, --domain <domain>     Custom domain for GitHub Pages (optional)
  -s, --source <dir>        Source directory (default: parent of script directory)
  -t, --target <dir>        Target directory (default: sibling of source dir with repo name)
      --desc <description>  GitHub repository description
      --workflow            Force GitHub Pages build type to 'workflow' (GitHub Actions)
      --legacy              Force GitHub Pages build type to 'legacy' (branch: main, path: /)
  -m, --move                Move files from source instead of copying
  -h, --help                Show this help message

Positional arguments:
  1: REPO_NAME
  2: CUSTOM_DOMAIN
  3: TARGET_DIR
  4: SOURCE_DIR
EOF
}

REPO_NAME=""
CUSTOM_DOMAIN=""
SOURCE_DIR=""
TARGET_DIR=""
REPO_DESC=""
BUILD_TYPE=""
MOVE_FILES=false

POSITIONAL_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      show_help
      exit 0
      ;;
    -r|--repo)
      REPO_NAME="$2"
      shift 2
      ;;
    -d|--domain)
      CUSTOM_DOMAIN="$2"
      shift 2
      ;;
    -s|--source)
      SOURCE_DIR="$2"
      shift 2
      ;;
    -t|--target)
      TARGET_DIR="$2"
      shift 2
      ;;
    --desc)
      REPO_DESC="$2"
      shift 2
      ;;
    --workflow)
      BUILD_TYPE="workflow"
      shift
      ;;
    --legacy)
      BUILD_TYPE="legacy"
      shift
      ;;
    -m|--move)
      MOVE_FILES=true
      shift
      ;;
    -*)
      echo "Unknown option: $1" >&2
      show_help
      exit 1
      ;;
    *)
      POSITIONAL_ARGS+=("$1")
      shift
      ;;
  esac
done

# Fill missing variables from positional arguments
if [ -z "$REPO_NAME" ] && [ ${#POSITIONAL_ARGS[@]} -ge 1 ]; then
  REPO_NAME="${POSITIONAL_ARGS[0]}"
fi
if [ -z "$CUSTOM_DOMAIN" ] && [ ${#POSITIONAL_ARGS[@]} -ge 2 ]; then
  CUSTOM_DOMAIN="${POSITIONAL_ARGS[1]}"
fi
if [ -z "$TARGET_DIR" ] && [ ${#POSITIONAL_ARGS[@]} -ge 3 ]; then
  TARGET_DIR="${POSITIONAL_ARGS[2]}"
fi
if [ -z "$SOURCE_DIR" ] && [ ${#POSITIONAL_ARGS[@]} -ge 4 ]; then
  SOURCE_DIR="${POSITIONAL_ARGS[3]}"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Resolve default SOURCE_DIR
if [ -z "$SOURCE_DIR" ]; then
  if [ "$(basename "$SCRIPT_DIR")" = "scripts" ]; then
    SOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
  else
    SOURCE_DIR="$SCRIPT_DIR"
  fi
else
  SOURCE_DIR="$(cd "$SOURCE_DIR" && pwd)"
fi

# Fallback for REPO_NAME if not supplied
if [ -z "$REPO_NAME" ]; then
  REPO_NAME="$(basename "$SOURCE_DIR")"
fi

# Resolve default TARGET_DIR
if [ -z "$TARGET_DIR" ]; then
  TARGET_DIR="$(dirname "$SOURCE_DIR")/$REPO_NAME"
fi

# Default repository description
if [ -z "$REPO_DESC" ]; then
  if [ -n "$CUSTOM_DOMAIN" ]; then
    REPO_DESC="$REPO_NAME project ($CUSTOM_DOMAIN)"
  else
    REPO_DESC="$REPO_NAME project"
  fi
fi

echo "=================================================="
echo " Setting up GitHub Pages Repository: $REPO_NAME"
echo " Source Directory:  $SOURCE_DIR"
echo " Target Directory:  $TARGET_DIR"
if [ -n "$CUSTOM_DOMAIN" ]; then
  echo " Custom Domain:     $CUSTOM_DOMAIN"
fi
echo " Description:       $REPO_DESC"
echo "=================================================="

# Check requirements
command -v gh >/dev/null 2>&1 || { echo "Error: 'gh' CLI is required."; exit 1; }
command -v git >/dev/null 2>&1 || { echo "Error: 'git' is required."; exit 1; }
command -v rsync >/dev/null 2>&1 || { echo "Error: 'rsync' is required."; exit 1; }

# Verify gh authentication
if ! gh auth status >/dev/null 2>&1; then
  echo "Error: gh CLI is not authenticated. Run 'gh auth login' first."
  exit 1
fi

GH_USER=$(gh api user -q .login)
echo "Authenticated as GitHub user: $GH_USER"

# Sync files if source and target differ
if [ "$SOURCE_DIR" != "$TARGET_DIR" ]; then
  if [ -d "$TARGET_DIR" ] && [ "$(ls -A "$TARGET_DIR" 2>/dev/null)" ]; then
    read -rp "Directory '$TARGET_DIR' already exists and is not empty. Overwrite/merge contents? (y/N): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
      echo "Aborted."
      exit 0
    fi
  else
    mkdir -p "$TARGET_DIR"
  fi

  echo "--> Syncing project files from $SOURCE_DIR to $TARGET_DIR..."
  rsync -a \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='dist' \
    --exclude='.astro' \
    --exclude='.venv' \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    --exclude='scratch' \
    --exclude='*:Zone.Identifier' \
    --exclude='.DS_Store' \
    "$SOURCE_DIR/" "$TARGET_DIR/"

  if [ "$MOVE_FILES" = true ]; then
    echo "--> Moving files: removing source files from $SOURCE_DIR..."
    find "$SOURCE_DIR" -mindepth 1 -maxdepth 1 \
      -not -name '.git' \
      -not -name 'scripts' \
      -exec rm -rf {} +
  fi
else
  echo "--> Source and target are identical ($TARGET_DIR). Skipping file sync."
fi

# Set up CNAME if custom domain provided
if [ -n "$CUSTOM_DOMAIN" ]; then
  echo "--> Writing CNAME file for '$CUSTOM_DOMAIN'..."
  echo "$CUSTOM_DOMAIN" > "$TARGET_DIR/CNAME"
  if [ -d "$TARGET_DIR/public" ]; then
    echo "$CUSTOM_DOMAIN" > "$TARGET_DIR/public/CNAME"
  fi
fi

# Ensure a README exists
if [ ! -f "$TARGET_DIR/README.md" ]; then
  echo "--> Generating initial README.md..."
  cat <<EOF > "$TARGET_DIR/README.md"
# $REPO_NAME

$REPO_DESC

Hosted on GitHub Pages at [https://${CUSTOM_DOMAIN:-$GH_USER.github.io/$REPO_NAME}](https://${CUSTOM_DOMAIN:-$GH_USER.github.io/$REPO_NAME}).
EOF
fi

# Initialize git repository
cd "$TARGET_DIR"

if [ ! -d ".git" ]; then
  echo "--> Initializing git repository..."
  git init -b main
else
  echo "--> Git repository already exists."
fi

git add .
git commit -m "feat: initial commit for $REPO_NAME" || echo "No changes to commit"

# Create repo on GitHub if it doesn't exist
echo "--> Creating or verifying GitHub repository '$GH_USER/$REPO_NAME'..."
if gh repo view "$GH_USER/$REPO_NAME" >/dev/null 2>&1; then
  echo "Repository '$GH_USER/$REPO_NAME' already exists on GitHub."
  git remote set-url origin "git@github.com:$GH_USER/$REPO_NAME.git" 2>/dev/null || \
  git remote add origin "git@github.com:$GH_USER/$REPO_NAME.git"
else
  gh repo create "$REPO_NAME" --public --source=. --remote=origin --description "$REPO_DESC"
fi

echo "--> Pushing 'main' branch to origin..."
git push -u origin main

# Detect Pages build type if not specified
if [ -z "$BUILD_TYPE" ]; then
  if [ -f "$TARGET_DIR/.github/workflows/deploy.yml" ] || grep -rq "actions/deploy-pages" "$TARGET_DIR/.github/workflows" 2>/dev/null; then
    BUILD_TYPE="workflow"
  else
    BUILD_TYPE="legacy"
  fi
fi

echo "--> Configuring GitHub Pages (build type: $BUILD_TYPE)..."
if [ "$BUILD_TYPE" = "workflow" ]; then
  gh api --method POST "repos/$GH_USER/$REPO_NAME/pages" \
    -f "build_type=workflow" 2>/dev/null || \
  gh api --method PUT "repos/$GH_USER/$REPO_NAME/pages" \
    -f "build_type=workflow" 2>/dev/null || true
else
  gh api --method POST "repos/$GH_USER/$REPO_NAME/pages" \
    -f "source[branch]=main" \
    -f "source[path]=/" 2>/dev/null || \
  gh api --method PUT "repos/$GH_USER/$REPO_NAME/pages" \
    -f "source[branch]=main" \
    -f "source[path]=/" 2>/dev/null || true
fi

# Set custom domain in GitHub Pages if provided
if [ -n "$CUSTOM_DOMAIN" ]; then
  echo "--> Setting custom domain to '$CUSTOM_DOMAIN'..."
  gh api --method PUT "repos/$GH_USER/$REPO_NAME/pages" \
    -f "cname=$CUSTOM_DOMAIN" 2>/dev/null || true
fi

echo "--> Fetching Pages configuration status..."
gh api "repos/$GH_USER/$REPO_NAME/pages" --jq '{url: .html_url, status: .status, cname: .cname, build_type: .build_type}' 2>/dev/null || true

echo "=================================================="
echo "✅ Repository $GH_USER/$REPO_NAME setup complete!"
if [ -n "$CUSTOM_DOMAIN" ]; then
  echo ""
  echo "Next Steps for Custom Domain ($CUSTOM_DOMAIN):"
  echo "1. In DNS (e.g. Cloudflare), add a CNAME record:"
  echo "   Name:   $(echo "$CUSTOM_DOMAIN" | cut -d'.' -f1)"
  echo "   Target: $GH_USER.github.io"
fi
echo "=================================================="
