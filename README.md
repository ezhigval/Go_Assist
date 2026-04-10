# Go_Assist (Modulr)

> AI-driven modular automation platform · Go + React + Python

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://go.dev)
[![Docs](https://img.shields.io/badge/Docs-RU%2FEN%2FZH-brightgreen)](./docs)

</div>

---

## Select Language / Select Language / Select Language

| Russian | English | Chinese |
|-------------|------------|---------|
| [Start](./docs/i18n/ru/README.md) | [Get Started](./docs/i18n/en/README.md) | [Start](./docs/i18n/zh/README.md) |

---

## Repository Structure

```
cmd/           # Entry points (modulr, telegram-bot)
core/          # Core: EventBus, Orchestrator, AI Engine
docs/          # Documentation (multilingual)
  i18n/
    ru/        # Russian (primary)
    en/        # English
    zh/        # Chinese
  shared/      # Images, diagrams
  nav/         # Navigation
modules/       # Domain modules (finance, calendar...)
config/        # Configurations
scripts/       # Utilities and validation
```

**Important**: All documentation is in `docs/i18n/`. Root files are for code and configs only.

---

## Quick Start

```bash
git clone https://github.com/ezhigval/Go_Assist.git
cd Go_Assist
go mod tidy
cp config/config.example.yaml config/config.yaml
go run cmd/modulr/main.go
```

:information_source: **Full instructions**: Russian | English | Chinese

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

:bulb: **All documentation edits should be made ONLY in** `docs/i18n/{ru,en,zh}/`. **Root .md files editing is prohibited.**`
