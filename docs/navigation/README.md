---
title: "Navigation Structure | Navigation Structure | Navigation Structure"
lang: [en, ru, zh]
version: v0.2
lastUpdated: 2026-04-10
---

# Navigation Structure | Navigation Structure | Navigation Structure

This directory contains navigation configuration for all documentation languages.

## Files

- `sidebars.json` - Docusaurus sidebar configuration
- `navbar.json` - Navigation bar configuration
- `toc.yaml` - Table of contents mapping

## Navigation Hierarchy

```
docs/
  README.md (Main index)
  i18n/
    ru/
      README.md (Russian index)
      architecture/
        README.md
        core-layer.md
        data-flow.md
      concepts/
        README.md
        actions.md
        results.md
      modules/
        README.md
        development.md
        examples.md
      ai/
        README.md
        planning.md
        reflection.md
    en/
      README.md (English index)
      [same structure as ru/]
    zh/
      README.md (Chinese index)
      [same structure as ru/]
```

## Breadcrumb Trail

Every page should include:
`Home > Section > Subsection > Current Page`

## Cross-language References

Use relative paths with language prefixes:
- `[English Version](../en/architecture/core-layer.md)`
- `[Russian Version](../ru/architecture/core-layer.md)`
- `[Chinese Version](../zh/architecture/core-layer.md)`
