#!/bin/bash

# Documentation Validation Script
# Validates links, spelling, Mermaid diagrams, and i18n synchronization

set -e

echo "=== Documentation Validation ==="
echo "Started at $(date)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}SUCCESS${NC}: $message"
            ;;
        "warning")
            echo -e "${YELLOW}WARNING${NC}: $message"
            ;;
        "error")
            echo -e "${RED}ERROR${NC}: $message"
            ;;
        "info")
            echo -e "INFO: $message"
            ;;
    esac
}

# 1. Check if required tools are installed
check_dependencies() {
    print_status "info" "Checking dependencies..."
    
    local missing_deps=()
    
    if ! command -v npx &> /dev/null; then
        missing_deps+=("npx")
    fi
    
    if ! command -v python3 &> /dev/null; then
        missing_deps+=("python3")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_status "error" "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
    
    print_status "success" "All dependencies found"
}

# 2. Link validation
validate_links() {
    print_status "info" "Validating links..."
    
    # Create link checker config if not exists
    if [ ! -f ".link-checker.json" ]; then
        cat > .link-checker.json << EOF
{
  "ignorePatterns": [
    {
      "pattern": "^http"
    }
  ],
  "replacementPatterns": [],
  "httpHeaders": [],
  "fallbackRetryCount": 2,
  "aliveStatusCodes": [200, 403]
}
EOF
    fi
    
    # Run link checker
    if npx markdown-link-check -c .link-checker.json docs/**/*.md; then
        print_status "success" "All links are valid"
    else
        print_status "error" "Link validation failed"
        return 1
    fi
}

# 3. Spell checking
validate_spelling() {
    print_status "info" "Checking spelling..."
    
    # Create cspell config if not exists
    if [ ! -f ".cspell.json" ]; then
        cat > .cspell.json << EOF
{
  "version": "0.2",
  "language": "en,ru",
  "words": [
    "EventBus",
    "Orchestrator", 
    "Docusaurus",
    "Mermaid",
    "Go",
    "golang",
    "struct",
    "interface",
    "ctx",
    "yaml",
    "json",
    "api",
    "ai",
    "i18n",
    "ru",
    "zh",
    "en",
    "README",
    "TODO",
    "WIP",
    "API",
    "URL",
    "HTTP",
    "HTTPS",
    "JSON",
    "YAML",
    "XML",
    "SQL",
    "NoSQL",
    "CRUD",
    "REST",
    "GraphQL",
    "JWT",
    "OAuth",
    "TLS",
    "SSL",
    "TCP",
    "UDP",
    "IP",
    "DNS",
    "CDN",
    "CI",
    "CD",
    "DevOps",
    "SaaS",
    "PaaS",
    "IaaS",
    "UX",
    "UI",
    "QA",
    "QA",
    "KPI",
    "SLA",
    "SLO",
    "NPM",
    "npm",
    "git",
    "GitHub",
    "GitLab",
    "Bitbucket",
    "Docker",
    "Kubernetes",
    "K8s",
    "Redis",
    "PostgreSQL",
    "MySQL",
    "MongoDB",
    "Linux",
    "Unix",
    "Windows",
    "macOS",
    "iOS",
    "Android",
    "JavaScript",
    "TypeScript",
    "Python",
    "Java",
    "C++",
    "C#",
    "Ruby",
    "PHP",
    "Swift",
    "Kotlin",
    "Rust",
    "Go",
    "Scala",
    "Perl",
    "Lua",
    "Bash",
    "Shell",
    "PowerShell",
    "CSS",
    "HTML",
    "SVG",
    "PNG",
    "JPG",
    "JPEG",
    "GIF",
    "PDF",
    "CSV",
    "TXT",
    "MD",
    "MDX",
    "YAML",
    "JSON",
    "XML",
    "SQL",
    "NoSQL",
    "CRUD",
    "REST",
    "GraphQL",
    "JWT",
    "OAuth",
    "TLS",
    "SSL",
    "TCP",
    "UDP",
    "IP",
    "DNS",
    "CDN",
    "CI",
    "CD",
    "DevOps",
    "SaaS",
    "PaaS",
    "IaaS",
    "UX",
    "UI",
    "QA",
    "KPI",
    "SLA",
    "SLO",
    "NPM",
    "git",
    "GitHub",
    "GitLab",
    "Bitbucket",
    "Docker",
    "Kubernetes",
    "K8s",
    "Redis",
    "PostgreSQL",
    "MySQL",
    "MongoDB",
    "Linux",
    "Unix",
    "Windows",
    "macOS",
    "iOS",
    "Android",
    "JavaScript",
    "TypeScript",
    "Python",
    "Java",
    "C++",
    "C#",
    "Ruby",
    "PHP",
    "Swift",
    "Kotlin",
    "Rust",
    "Go",
    "Scala",
    "Perl",
    "Lua",
    "Bash",
    "Shell",
    "PowerShell",
    "CSS",
    "HTML",
    "SVG",
    "PNG",
    "JPG",
    "JPEG",
    "GIF",
    "PDF",
    "CSV",
    "TXT",
    "MD",
    "MDX"
  ],
  "ignoreRegExpList": [
    "/\\b[A-Z]{2,}\\b/g",
    "/\\b0x[a-fA-F0-9]+\\b/g",
    "/\\b\\d+\\.\\d+\\.\\d+\\b/g",
    "/\\b[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\\b/g"
  ],
  "ignorePaths": [
    "node_modules/**",
    ".git/**",
    "dist/**",
    "build/**"
  ]
}
EOF
    fi
    
    # Run spell check
    if npx cspell --config .cspell.json "docs/**/*.{md,mdx}"; then
        print_status "success" "Spell check passed"
    else
        print_status "warning" "Spell check found issues"
        return 1
    fi
}

# 4. Mermaid validation
validate_mermaid() {
    print_status "info" "Validating Mermaid diagrams..."
    
    # Find all .md files with mermaid blocks
    local mermaid_files=$(find docs -name "*.md" -exec grep -l "```mermaid" {} \;)
    
    if [ -z "$mermaid_files" ]; then
        print_status "info" "No Mermaid diagrams found"
        return 0
    fi
    
    local mermaid_errors=0
    
    for file in $mermaid_files; do
        # Extract mermaid blocks and validate syntax
        local mermaid_blocks=$(sed -n '/```mermaid/,/```/p' "$file" | grep -v "```")
        
        if [ -n "$mermaid_blocks" ]; then
            # Basic syntax validation
            echo "$mermaid_blocks" | grep -E "^(graph|flowchart|sequenceDiagram|classDiagram|stateDiagram|erDiagram|journey|gantt|pie|gitgraph)" > /dev/null
            if [ $? -ne 0 ]; then
                print_status "error" "Invalid Mermaid syntax in $file"
                mermaid_errors=$((mermaid_errors + 1))
            fi
        fi
    done
    
    if [ $mermaid_errors -eq 0 ]; then
        print_status "success" "All Mermaid diagrams are valid"
    else
        print_status "error" "Found $mermaid_errors Mermaid diagram errors"
        return 1
    fi
}

# 5. i18n synchronization check
check_i18n_sync() {
    print_status "info" "Checking i18n synchronization..."
    
    # Create Python script for sync checking
    cat > docs/scripts/check-i18n-sync.py << 'EOF'
#!/usr/bin/env python3
import os
import yaml
import json
from pathlib import Path

def load_yaml(file_path):
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            return yaml.safe_load(f)
    except:
        return {}

def check_file_structure():
    docs_path = Path("docs/i18n")
    languages = ["ru", "en", "zh"]
    sections = ["architecture", "concepts", "modules", "ai"]
    
    issues = []
    
    for section in sections:
        section_files = {}
        for lang in languages:
            lang_path = docs_path / lang / section
            if lang_path.exists():
                files = [f.name for f in lang_path.glob("*.md")]
                section_files[lang] = set(files)
        
        # Check if all languages have same files
        if section_files:
            reference = section_files[languages[0]]
            for lang in languages[1:]:
                if lang in section_files:
                    missing = reference - section_files[lang]
                    extra = section_files[lang] - reference
                    if missing:
                        issues.append(f"{section}/{lang}: Missing files: {missing}")
                    if extra:
                        issues.append(f"{section}/{lang}: Extra files: {extra}")
    
    return issues

def check_frontmatter():
    docs_path = Path("docs/i18n")
    languages = ["ru", "en", "zh"]
    
    issues = []
    
    for lang in languages:
        lang_path = docs_path / lang
        if not lang_path.exists():
            continue
            
        for md_file in lang_path.rglob("*.md"):
            try:
                with open(md_file, 'r', encoding='utf-8') as f:
                    content = f.read()
                    
                # Check frontmatter
                if content.startswith('---'):
                    frontmatter_end = content.find('---', 3)
                    if frontmatter_end == -1:
                        issues.append(f"{md_file}: Unclosed frontmatter")
                    else:
                        frontmatter = content[3:frontmatter_end]
                        try:
                            yaml.safe_load(frontmatter)
                        except yaml.YAMLError as e:
                            issues.append(f"{md_file}: Invalid frontmatter: {e}")
            except Exception as e:
                issues.append(f"{md_file}: Error reading file: {e}")
    
    return issues

def main():
    issues = []
    
    issues.extend(check_file_structure())
    issues.extend(check_frontmatter())
    
    if issues:
        print("i18n synchronization issues:")
        for issue in issues:
            print(f"  - {issue}")
        return 1
    else:
        print("i18n synchronization OK")
        return 0

if __name__ == "__main__":
    exit(main())
EOF
    
    # Run sync check
    if python3 docs/scripts/check-i18n-sync.py; then
        print "SUCCESS: i18n synchronization OK"
    else
        print "WARNING: i18n synchronization issues found"
        return 1
    fi
}

# 6. Check required files
check_required_files() {
    print_status "info" "Checking required files..."
    
    local required_files=(
        "docs/TEMPLATE.md"
        "docs/i18n/glossary.yaml"
        "docs/navigation/sidebars.json"
        "docs/shared/README.md"
    )
    
    local missing_files=()
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            missing_files+=("$file")
        fi
    done
    
    if [ ${#missing_files[@]} -ne 0 ]; then
        print_status "error" "Missing required files: ${missing_files[*]}"
        return 1
    fi
    
    print_status "success" "All required files present"
}

# Main execution
main() {
    local exit_code=0
    
    check_dependencies || exit_code=1
    validate_links || exit_code=1
    validate_spelling || exit_code=1
    validate_mermaid || exit_code=1
    check_i18n_sync || exit_code=1
    check_required_files || exit_code=1
    
    echo ""
    echo "=== Validation Complete ==="
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All validations passed!"
    else
        print_status "error" "Some validations failed. Please fix issues before committing."
    fi
    echo "Completed at $(date)"
    
    exit $exit_code
}

# Run main function
main "$@"
