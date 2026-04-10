#!/bin/bash
set -e

echo "Validating documentation..."

# 1. Check broken links
echo "Checking links..."
npx markdown-link-check -c .github/link-checker.json "docs/**/*.md" 2>/dev/null || echo "Link check warnings (non-blocking)"

# 2. Check spelling (only for ru/ and en/)
echo "Checking spelling..."
npx cspell --config .cspell.json "docs/i18n/{ru,en}/**/*.md" || echo "Spelling warnings"

# 3. Check frontmatter
echo "Checking frontmatter..."
grep -L "^---" docs/**/*.md && echo "Files missing frontmatter!" && exit 1 || echo "All files have frontmatter"

# 4. Check for translit in Russian docs
echo "Checking for translit in Russian docs..."
if grep -r "[a-zA-Z]\{5,\}" docs/i18n/ru/ --include="*.md" | grep -v "```" | grep -v "code" | grep -v "EventBus\|Orchestrator\|AI\|API"; then
  echo "Found potential translit in Russian docs!"
  exit 1
else
  echo "No translit found in Russian docs"
fi

# 5. Check Mermaid
echo "Checking Mermaid syntax..."
grep -l "```mermaid" docs/**/*.md | xargs -I {} sh -c 'grep -A 20 "```mermaid" "{}" | grep -q "```" || (echo "Unclosed Mermaid in {}" && exit 1)'

echo "Documentation validation passed!"
