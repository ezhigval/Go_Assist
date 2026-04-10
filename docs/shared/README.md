---
title: "Shared Resources | Shared Resources | Shared Resources"
lang: [en, ru, zh]
version: v0.2
lastUpdated: 2026-04-10
---

# Shared Resources | Shared Resources | Shared Resources

This directory contains shared resources used across all documentation languages.

## Directory Structure

```
shared/
  images/     # Images, screenshots, icons
  diagrams/    # Mermaid diagrams, flowcharts
  data/        # Configuration files, sample data
```

## Usage Guidelines

### Images
- Use descriptive filenames: `architecture-overview.png`
- Include alt text in markdown: `![Architecture Overview](../shared/images/architecture-overview.png)`
- Keep images under 2MB
- Use SVG format for diagrams when possible

### Diagrams
- Store Mermaid diagram definitions here for reuse
- Reference in docs: `![](../shared/diagrams/execution-flow.mmd)`
- Test diagrams in both GitHub and Docusaurus

### Data
- Configuration examples
- Sample JSON/YAML files
- Test data for examples

## Naming Conventions

- Use kebab-case for filenames
- Include language suffix if needed: `config-example-en.yaml`
- Keep names descriptive but concise
