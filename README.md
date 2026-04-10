[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://go.dev)
[![Status](https://img.shields.io/badge/Status-Design_Complete-yellow)](./ROADMAP.md)
[![Telegram](https://img.shields.io/badge/Contact-@ezhigval-2CA5E0)](https://t.me/ezhigval)

# Modulr (Go_Assist)
> **Конструктор персональной экосистемы. Собирай. Подключай. Масштабируй.**

Event-driven, context-aware, AI-orchestrated monorepo на Go + React + Python. Собирай персональный ассистент из LEGO-модулей: финансы, календарь, трекер, знания и другие. Каждый модуль работает в своей сфере жизни (personal, family, business, health), но связи между ними автоматически строятся через EventBus и AI.

---

## 2.0 Philosophy & Principles

- **LEGO-архитектура**: модули изолированы, общаются только через EventBus
- **Event-First**: нулевая связанность, лёгкое тестирование, горизонтальное масштабирование
- **Контекстная изоляция**: `personal` != `family` != `business`, но связи работают через AI
- **Гибридный AI**: OpenAI для MVP -> локальные модели для продакшена
- **Приватность по умолчанию**: данные не покидают сервер без явного согласия
- **Прогрессивный фронтенд**: Telegram Mini App -> PWA -> мобильные -> десктопы

---

## 3.0 Architecture

```
Frontend Layer
    |
Transport Layer (Telegram Bot API / HTTP / WebSocket)
    |
Core Layer (EventBus | Orchestrator | AI Engine | State)
    |
Domain Modules (finance | calendar | tracker | knowledge | ...)
    |
Data Layer (PostgreSQL | Redis | Vector DB | Local Storage)
```

**Ключевые компоненты:**
- `core/events/` - EventBus для системы
- `core/orchestrator/` - Валидация решений AI
- `core/aiengine/` - Реестры моделей и маршрутизация
- Domain модули - Изолированная бизнес-логика

**Документация:**
- [Проектные правила](./PROJECT_RULES.md)
- [Экосистема и модули](./ARCHITECTURE.md#-экосистема-модули)
- [AI-архитектура](./ai/AI_ARCHITECTURE.md)
- [Frontend-стандарты](./frontend/FRONTEND_RULES.md)

---

## 4.0 Ecosystem: Contexts × Operations

| Operation \ Context | personal | family | business | health | travel | pets |
|---------------------|----------|--------|----------|--------|--------|------|
| **finance** | бюджет, подписки | совместные траты | прибыль/расходы | страховки, БАДы | билеты, визы | ветклиника, корм |
| **calendar** | личное время | кружки, ужины | встречи, дедлайны | приёмы врача | вылеты, экскурсии | прививки, прогулки |
| **tracker** | привычки, цели | домашние дела | спринты, OKR | тренировки, диета | чек-листы сборов | уход, дрессировка |
| **knowledge** | дневник, идеи | рецепты, правила | регламенты, гайды | методики, симптомы | гиды, фразы | порода, рацион |
| **contacts** | друзья, эксперты | родственники, учителя | коллеги, клиенты | тренеры, врачи | гиды, попутчики | ветеринары, грумеры |

**Пример кросс-связи:**  
`Заметка: "Купить молоко по пути домой"` -> AI распознаёт интент ->  
`calendar/` ставит напоминание + `finance/` резервирует бюджет +  
`logistics/` строит маршрут через магазин -> все события в EventBus.

---

## 5.0 AI-Subsystem

**Гибридный режим:** Remote API (MVP) <-> Local Models (Prod)

| Component | Task | Technologies |
|-----------|------|-------------|
| AI Gateway | Маршрутизация запросов, PII-редакция | Go, gRPC, middleware |
| Model Registry | Реестры моделей, версионирование, health-checks | YAML config, Prometheus |
| Domain Services | Финансы, здоровье, логистика, знания | Python, FastAPI, scikit-learn, ONNX |
| Feedback Loop | Обучение на фидбеке, обновление confidence | Async queue, batch training |
| Vector Memory | Долгосрочный контекст, ассоциации | Chroma/Qdrant, embeddings |

**Безопасность:**
- Все внешние запросы проходят PII-редакцию
- `scope`-изоляция: данные `personal` не передаются в `business`
- `confidence < 0.7` -> требует подтверждения пользователя
- Логи без персональных данных, аудит всех решений

**Документация:**
- [AI Архитектура](./ai/AI_ARCHITECTURE.md)
- [AI Roadmap](./ai/AI_ROADMAP.md)
- [AI Правила](./ai/AI_RULES.md)

---

## 6.0 Frontend & Platforms

**Прогрессивное усиление:** один код -> все платформы

| Platform | Status | Technologies |
|----------|--------|-------------|
| Telegram Mini App | MVP | React, @twa-dev/sdk, Vite |
| PWA (Web) | В разработке | React, Vite PWA, IndexedDB |
| iOS / Android | Планируется | React Native + Capacitor |
| Desktop (Win/macOS/Linux) | Планируется | Tauri (Rust + React) |
| Wearables (watchOS/Wear OS) | Идея | Нативные компликации |

**Особенности:**
- Контекстная навигация: переключай сферы жизни в один клик
- Визуализация связей: карточки показывают связанные сущности
- Офлайн-первый: кэширование, синхронизация при появлении сети
- Модульный UI: компоненты = бэкенд-модули, переиспользование 90%+

**Документация:**
- [Frontend Правила](./frontend/FRONTEND_RULES.md)
- [Frontend Roadmap](./frontend/FRONTEND_ROADMAP.md)

---

## 7.0 Quick Start

### Требования
- Go 1.21+
- Node.js 18+ / npm 9+
- Docker + Docker Compose (опционально, для локального AI-стека)
- PostgreSQL 15+ (или используйте Supabase free tier)

### 1. Клонирование
```bash
git clone https://github.com/ezhigval/Go_Assist.git
cd Go_Assist
```

### 2. Настройка окружения
```bash
# Скопируй шаблоны конфигов
cp .env.example .env
cp config/config.example.yaml config/config.yaml

# Заполни переменные (минимум для локального запуска):
# TELEGRAM_TOKEN=your_bot_token
# DB_DSN=postgres://user:pass@localhost:5432/modulr?sslmode=disable
# AI_PROVIDER=openai  # или "local" для самохостинга
```

### 3. Запуск ядра (Go)
```bash
cd core
go mod tidy
go run main.go
# Ядро запустится в режиме polling, слушает EventBus
```

### 4. Запуск frontend (Telegram Mini App)
```bash
cd frontend
npm install
npm run dev:telegram
# Открой бота в Telegram -> нажми "Запустить веб-приложение"
```

### 5. Локальный AI-стек (опционально)
```bash
cd ai
docker compose -f docker-compose.local.yml up -d
# Запустит Ollama + FastAPI-сервисы для локального инференса
```

**Полная документация:**
- [Установка и настройка](#70-быстрый-старт)
- [Конфигурация](#70-быстрый-старт)
- [API Reference](./README.md)

---

## 8.0 Open Source & Community

**Modulr** - eto otkrytyy proyekt, kotoryy razvivayetsya blagodarya soobshchestvu.

### License
Код распространяется под лицензией MIT.  
Ты можешь:
- **Использовать** в личных и коммерческих проектах
- **Модифицировать** и форкать
- **Распространять** с изменениями
- **Не неси ответственности** за использование "как есть"

### Поддержка проекта
Разработка ведётся на энтузиазме. Любая помощь ускоряет развитие:
- **GitHub Sponsors**
- **Open Collective** (placeholder)
- **Crypto: USDT/TRC20** (placeholder)

**Средства идут на:**
- Серверы и инфраструктуру для демо/тестов
- Токены для внешних AI-API (на этапе MVP)
- Дизайн, документацию, переводы
- Оплату контрибьюторов за сложные задачи

### Присоединяйся к команде
Ищем энтузиастов для развития проекта:

| Роль | Задачи | Stack |
|------|---------|-------|
| **Go Backend** | Ядро, EventBus, модули, gRPC | Go, pgx, context, sync |
| **React Frontend** | UI, PWA, Telegram Mini App | React, TypeScript, Tailwind |
| **Python/AI** | Доменные модели, инференс, обучение | FastAPI, scikit-learn, ONNX |
| **DevOps** | Docker, CI/CD, monitoring, deploy | Docker, GH Actions, Prometheus |
| **Tech Writer** | Документация, туториалы, переводы | Markdown, Docusaurus |
| **QA / Testing** | Тесты, баг-репорты, юзабилити | Vitest, Playwright, manual |

**Условия:**
- **Удалённо**, гибкий график
- **Реальные production-задачи**, менторство
- **Влияние на архитектуру и роадмеп**
- **Признание в документации**, мерч, доля в премиум-модулях (опционально)

**Как начать:**
1. Изучи [PROJECT_RULES.md](./PROJECT_RULES.md) и [CONTRIBUTING.md](./CONTRIBUTTING.md)
2. Найди задачу с меткой `good first issue`
3. Напиши в [GitHub Discussions](https://github.com/ezhigval/Go_Assist/discussions) или в Telegram @ezhigval
4. Создай форк, сделай pull-request

---

## 9.0 Roadmap & Status

| Этап | Статус | Описание |
|------|--------|-----------|
| **Проектирование архитектуры** | **Complete** | Ядро, модули, AI, frontend |
| **Документация** | **Complete** | Правила, экосистема, роадмепы |
| **Прототип ядра** | **Complete** | EventBus, Orchestrator, контракты |
| **Реализация MVP** | **In Progress** | Telegram Mini App + 3 модуля (Q1 2025) |
| **Гибридный AI** | **Planned** | Локальные модели + feedback loop (Q2 2025) |
| **PWA + офлайн-режим** | **Planned** | (Q3 2025) |
| **Мобильные приложения** | **Planned** | iOS/Android (Q4 2025) |
| **Premium-модули** | **Planned** | Монетизация (2026) |

**Детальный план:**
- [Основной Roadmap](./ROADMAP.md)
- [AI Roadmap](./ai/AI_ROADMAP.md)
- [Frontend Roadmap](./frontend/FRONTEND_ROADMAP.md)

---

## 10.0 Contacts & Support

- **Основной контакт:** @ezhigval (Telegram)
- **Обсуждения:** [GitHub Discussions](https://github.com/ezhigval/Go_Assist/discussions)
- **Баг-репорты:** [GitHub Issues](https://github.com/ezhigval/Go_Assist/issues)
- **Email:** hello@modulr.dev (placeholder)
- **Сайт:** modulr.dev (placeholder)

---

<p align="center">
  <b>Не пиши приложения. Собирай их.</b><br><br>
  Modulr - инфраструктура для тех, кто ценит контроль, приватность и гибкость.
</p>
