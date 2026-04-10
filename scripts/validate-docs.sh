#!/bin/bash
set -e

echo "Validating documentation..."

# 1. Check broken links
echo "Checking links..."
npx markdown-link-check -c .github/link-checker.json "docs/**/*.md" 2>/dev/null || echo "Link check warnings"

# 2. Check spelling
echo "Checking spelling..."
npx cspell --config .cspell.json "docs/i18n/{ru,en}/**/*.md" || echo "Spelling warnings"

# 3. Check frontmatter
echo "Checking frontmatter..."
missing_files=$(find docs -name "*.md" -exec sh -c 'if ! head -1 "$1" | grep -q "^---"; then echo "$1"; fi' _ {} \;)
if [ -n "$missing_files" ]; then
  echo "Files missing frontmatter:"
  echo "$missing_files"
  exit 1
else
  echo "All files have frontmatter"
fi

# 4. Check for translit in Russian docs
echo "Checking for translit in Russian docs..."
translit_found=false
for file in docs/i18n/ru/*.md; do
  # Skip YAML frontmatter (between --- and ---)
  content=$(sed '/^---/,/^---/d' "$file")
  # Check for translit in content only, excluding code blocks, technical terms, headers, and descriptions
  if echo "$content" | grep -E '[a-zA-Z]{5,}' | grep -v 'EventBus|Orchestrator|AI|API|HTTP|Go|React' | grep -v '^```' | grep -v '^#' | grep -v '^>' | grep -v 'Language\|RU\|EN\|ZH' > /dev/null; then
    echo "Found potential translit in $file:"
    echo "$content" | grep -E '[a-zA-Z]{5,}' | grep -v 'EventBus|Orchestrator|AI|API|HTTP|Go|React' | grep -v '^```' | grep -v '^#' | grep -v '^>' | grep -v 'Language\|RU\|EN\|ZH' | head -3
    translit_found=true
  fi
done

if [ "$translit_found" = true ]; then
  exit 1
else
  echo "No translit found in Russian docs"
fi

echo "Documentation validation passed!"
