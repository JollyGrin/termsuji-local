#!/bin/bash
set -e

CHANGELOG_FILE="CHANGELOG.md"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get current version from latest tag
CURRENT=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo -e "${YELLOW}Current version:${NC} $CURRENT"

# Parse version components
VERSION=${CURRENT#v}
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

echo ""
echo "What type of release?"
echo "  1) patch  (v$MAJOR.$MINOR.$((PATCH + 1))) - bug fixes"
echo "  2) minor  (v$MAJOR.$((MINOR + 1)).0) - new features"
echo "  3) major  (v$((MAJOR + 1)).0.0) - breaking changes"
echo "  4) custom"
read -p "Select [1-4]: " BUMP_TYPE

case $BUMP_TYPE in
    1) NEW_VERSION="v$MAJOR.$MINOR.$((PATCH + 1))" ;;
    2) NEW_VERSION="v$MAJOR.$((MINOR + 1)).0" ;;
    3) NEW_VERSION="v$((MAJOR + 1)).0.0" ;;
    4) read -p "Enter version (e.g., v1.2.3): " NEW_VERSION ;;
    *) echo "Invalid selection"; exit 1 ;;
esac

echo ""
echo -e "${GREEN}New version:${NC} $NEW_VERSION"
echo ""

# Changelog entry
echo "Enter changelog notes (empty line to finish):"
NOTES=""
while IFS= read -r line; do
    [ -z "$line" ] && break
    NOTES="${NOTES}- ${line}\n"
done

# Update CHANGELOG.md
if [ ! -f "$CHANGELOG_FILE" ]; then
    echo "# Changelog" > "$CHANGELOG_FILE"
    echo "" >> "$CHANGELOG_FILE"
fi

# Prepend new entry
DATE=$(date +%Y-%m-%d)
{
    head -n 2 "$CHANGELOG_FILE"
    echo "## [$NEW_VERSION] - $DATE"
    echo ""
    echo -e "$NOTES"
    tail -n +3 "$CHANGELOG_FILE"
} > "${CHANGELOG_FILE}.tmp"
mv "${CHANGELOG_FILE}.tmp" "$CHANGELOG_FILE"

echo ""
echo -e "${GREEN}Updated $CHANGELOG_FILE${NC}"
echo ""

# Show summary
echo "=== Release Summary ==="
echo "Version: $NEW_VERSION"
echo "Changes:"
echo -e "$NOTES"
echo ""

read -p "Commit, tag, and push? [y/N]: " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo "Aborted. Changelog was updated but not committed."
    exit 0
fi

# Commit and tag
git add "$CHANGELOG_FILE"
git commit -m "Release $NEW_VERSION"
git tag "$NEW_VERSION"
BRANCH=$(git rev-parse --abbrev-ref HEAD)
git push origin "$BRANCH"
git push origin "$NEW_VERSION"

echo ""
echo -e "${GREEN}Released $NEW_VERSION${NC}"
echo "GitHub Actions will build and publish binaries."
echo "Check: https://github.com/JollyGrin/termsuji-local/actions"
