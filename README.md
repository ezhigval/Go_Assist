[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://go.dev)
[![Status](https://img.shields.io/badge/Status-Design_Complete-yellow)](./docs/ROADMAP.md)
[![Telegram](https://img.shields.io/badge/Contact-@ezhigval-2CA5E0)](https://t.me/ezhigval)

# Modulr (Go_Assist)
> **Konstruktor personal'noy ekosistemy. Sobiray. Podklyuchay. Masshtabiruy.**

Event-driven, context-aware, AI-orchestrated monorepo na Go + React + Python. Sobiray personal'nyy assistent iz LEGO-modulney: finansy, kalendar', tracker, znaniya i drugie. Kazhdyy moduley rabotayet v svoey sfere zhizni (personal, family, business, health), no svyazy mezhdu nimi avtomaticheski stroitsya cherez EventBus i AI.

---

## 2.0 Philosophy & Principles

- **LEGO-arkhitektura**: moduli izolirovany, obshchayutsya tol'ko cherez EventBus
- **Event-First**: nulevaya svyazannost', legkoye testirovaniye, gorizontal'noye masshtabirovaniye
- **Kontekstnaya izolyatsiya**: `personal` != `family` != `business`, no svyazi rabotayut cherez AI
- **Gibridnyy AI**: OpenAI dlya MVP -> lokal'nyye modeli dlya prodakshena
- **Privatnost' po umolchaniyu**: dannyye ne pokidayut server bez yavnogo soglasiya
- **Progressivnyy frontend**: Telegram Mini App -> PWA -> mobil'nyye -> desktopy

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

**Klyuchevyye komponenty:**
- `core/events/` - EventBus dlya sistemy
- `core/orchestrator/` - Validatsiya resheniy AI
- `core/aiengine/` - Reestry modeley i marshrutizatsiya
- Domain moduli - Izolirovannaya biznes-logika

**Dokumentatsiya:**
- [Proyektmyye pravila](./docs/PROJECT_RULES.md)
- [Ekosistema i moduli](./docs/ECOSYSTEM_DESIGN.yaml)
- [AI-arkhitektura](./ai/AI_ARCHITECTURE.md)
- [Frontend-standarty](./frontend/FRONTEND_RULES.md)

---

## 4.0 Ecosystem: Contexts × Operations

| Operation \ Context | personal | family | business | health | travel | pets |
|---------------------|----------|--------|----------|--------|--------|------|
| **finance** | byudzhet, podpiski | sovmestnyye traty | pribyl'/raskhody | strakhovki, BADy | bilety, vizy | vetklinika, korm |
| **calendar** | lichnoye vremya | krugki, uzhiny | vstrechi, dedlayny | priyomy vracha | vylety, ekskursii | privivki, progulki |
| **tracker** | privychki, tseli | domashniye dela | sprinty, OKR | trenirovki, dieta | chek-listy sborov | ukhod, dressirovka |
| **knowledge** | dnevnik, idei | retsepty, pravila | reglamenty, gaydy | metodiki, simptomy | gidy, frazy | poroda, ratsion |
| **contacts** | druz'ya, ekspertry | rodstvenniki, uchitelya | kollegi, kliyenty | trenery, vrachi | gidy, poputchiki | vety, grumery |

**Primer kross-svyazi:**  
`Zametka: "Kupit' moloko po puti domoy"` -> AI raspoznavayet intent ->  
`calendar/` stavit napominaniye + `finance/` rezerviruyet byudzhet +  
`logistics/`stroyit marshrut cherez magazin -> vsye sobytiya v EventBus.

---

## 5.0 AI-Subsystem

**Gibridnyy rezhim:** Remote API (MVP) <-> Local Models (Prod)

| Component | Task | Technologies |
|-----------|------|-------------|
| AI Gateway | Marshrutizatsiya zaprosov, PII-redaktsiya | Go, gRPC, middleware |
| Model Registry | Reestry modeley, versiirovaniye, health-checks | YAML config, Prometheus |
| Domain Services | Finansy, zdorov'ye, logistika, znaniya | Python, FastAPI, scikit-learn, ONNX |
| Feedback Loop | Obucheniye na fidbeke, obnovleniye confidence | Async queue, batch training |
| Vector Memory | Dolgosrochnyy kontekst, assotsiatsii | Chroma/Qdrant, embeddings |

**Bezopasnost':**
- Vse vneshniye zaprosy prokhodyat PII-redaktsiyu
- `scope`-izolyatsiya: dannyye `personal` ne peredayutsya v `business`
- `confidence < 0.7` -> trebuyet podtverzhdeniya pol'zovatelya
- Logi bez personal'nykh dannykh, audit vsekh resheniy

**Dokumentatsiya:**
- [AI Arkhitektura](./ai/AI_ARCHITECTURE.md)
- [AI Roadmap](./ai/AI_ROADMAP.md)
- [AI Pravila](./ai/AI_RULES.md)

---

## 6.0 Frontend & Platforms

**Progressivnoye usileniye:** odin kod -> vse platformy

| Platform | Status | Technologies |
|----------|--------|-------------|
| Telegram Mini App | MVP | React, @twa-dev/sdk, Vite |
| PWA (Web) | V razrabotke | React, Vite PWA, IndexedDB |
| iOS / Android | Planiruyetsya | React Native + Capacitor |
| Desktop (Win/macOS/Linux) | Planiruyetsya | Tauri (Rust + React) |
| Wearables (watchOS/Wear OS) | Ideya | Nativnyye komplikatsii |

**Osobennosti:**
- Kontekstnaya navigatsiya: pereklyuchay sfery zhizni v odin klik
- Vizualizatsiya svyazey: kartochki pokazyvayut svyazannyye sushchnosti
- Oflayn-pervyy: keshirovaniye, sinkhronizatsiya pri poyavlenii seti
- Modul'nyy UI: komponenty = backend-moduli, pereispol'zovaniye 90%+

**Dokumentatsiya:**
- [Frontend Pravila](./frontend/FRONTEND_RULES.md)
- [Frontend Roadmap](./frontend/FRONTEND_ROADMAP.md)

---

## 7.0 Quick Start

### Trebovaniya
- Go 1.21+
- Node.js 18+ / npm 9+
- Docker + Docker Compose (optsional'no, dlya lokall'nogo AI-steka)
- PostgreSQL 15+ (ili ispol'zuyte Supabase free tier)

### 1. Klonirovaniye
```bash
git clone https://github.com/ezhigval/Go_Assist.git
cd Go_Assist
```

### 2. Nastroyka okruzheniya
```bash
# Skopiruy shablony konfigigov
cp .env.example .env
cp config/config.example.yaml config/config.yaml

# Zapolni peremennyye (minimum dlya lokal'nogo zapuska):
# TELEGRAM_TOKEN=your_bot_token
# DB_DSN=postgres://user:pass@localhost:5432/modulr?sslmode=disable
# AI_PROVIDER=openai  # ili "local" dlya samokhostinga
```

### 3. Zapusk yadra (Go)
```bash
cd core
go mod tidy
go run main.go
# Yadro zapustitsya v rezhime polling, slushayet EventBus
```

### 4. Zapusk frontend (Telegram Mini App)
```bash
cd frontend
npm install
npm run dev:telegram
# Otkroy bota v Telegram -> nazhmi "Zapustit' veb-prilozheniye"
```

### 5. Lokal'nyy AI-stek (optsional'no)
```bash
cd ai
docker compose -f docker-compose.local.yml up -d
# Zapustit Ollama + FastAPI-servisy dlya lokal'nogo inferensa
```

**Polnaya dokumentatsiya:**
- [Ustanovka i nastroyka](./docs/INSTALLATION.md)
- [Konfiguratsiya](./docs/CONFIGURATION.md)
- [API Reference](./docs/API.md)

---

## 8.0 Open Source & Community

**Modulr** - eto otkrytyy proyekt, kotoryy razvivayetsya blagodarya soobshchestvu.

### License
Kod rasprostranyayetsya pod litsenziyey MIT.  
Ty mozhesh:
- **Ispol'zovat'** v lichnykh i kommercheskikh proyektakh
- **Modifitsirovat'** i forkat'
- **Rasprostranyat'** s izmeneniyami
- **Ne nesi otvetstvennosti** za ispol'zovaniye "kak yest'"

### Podderzhka proyekta
Razrabotka vedyotsya na entuziazme. Lyubaya pomoshch' uskoryayet razvitiye:
- **GitHub Sponsors**
- **Open Collective** (placeholder)
- **Crypto: USDT/TRC20** (placeholder)

**Sredstva idut na:**
- Servery i infrastrukturu dlya demo/testov
- Tokeny dlya vneshnikh AI-API (na etape MVP)
- Dizayn, dokumentatsiyu, perevody
- Oplatu kontrib'yutorov za slozhnyye zadachi

### Prisoedinysya k komande
Ishchem entuziastov dlya razvitiya proyekta:

| Rol' | Zadachi | Stack |
|------|---------|-------|
| **Go Backend** | Yadro, EventBus, moduli, gRPC | Go, pgx, context, sync |
| **React Frontend** | UI, PWA, Telegram Mini App | React, TypeScript, Tailwind |
| **Python/AI** | Domennyye modeli, inferens, obucheniye | FastAPI, scikit-learn, ONNX |
| **DevOps** | Docker, CI/CD, monitoring, deploy | Docker, GH Actions, Prometheus |
| **Tech Writer** | Dokumentatsiya, tutoryaly, perevody | Markdown, Docusaurus |
| **QA / Testing** | Testy, bag-reporty, yuzabiliti | Vitest, Playwright, manual |

**Usloviya:**
- **Udalonno**, gibkiy grafik
- **Real'nyye production-zadachi**, mentorstvo
- **Vliyaniye na arkhitekturu i roadmep**
- **Priznaniye v dokumentatsii**, merch, dolya v premium-modulyakh (optsional'no)

**Kak nachat':**
1. Izuchi [PROJECT_RULES.md](./docs/PROJECT_RULES.md) i [CONTRIBUTING.md](./docs/CONTRIBUTING.md)
2. Naydi zadachu s metkoy `good first issue`
3. Napishi v [GitHub Discussions](https://github.com/ezhigval/Go_Assist/discussions) ili v Telegram @ezhigval
4. Sozdai fork, sdelay pull-request

---

## 9.0 Roadmap & Status

| Etap | Status | Opisaniye |
|------|--------|-----------|
| **Proektirovaniye arkhitektury** | **Complete** | Yadro, moduli, AI, frontend |
| **Dokumentatsiya** | **Complete** | Pravila, ekosistema, roadmepy |
| **Prototip yadra** | **Complete** | EventBus, Orchestrator, kontrakty |
| **Realizatsiya MVP** | **In Progress** | Telegram Mini App + 3 modulya (Q1 2025) |
| **Gibridnyy AI** | **Planned** | Lokal'nyye modeli + feedback loop (Q2 2025) |
| **PWA + oflayn-rezhim** | **Planned** | (Q3 2025) |
| **Mobil'nyye prilozheniya** | **Planned** | iOS/Android (Q4 2025) |
| **Premium-moduli** | **Planned** | Monetizatsiya (2026) |

**Detal'nyy plan:**
- [Osnovnoy Roadmap](./docs/ROADMAP.md)
- [AI Roadmap](./ai/AI_ROADMAP.md)
- [Frontend Roadmap](./frontend/FRONTEND_ROADMAP.md)

---

## 10.0 Contacts & Support

- **Osnovnoy kontakt:** @ezhigval (Telegram)
- **Obsuzhdeniya:** [GitHub Discussions](https://github.com/ezhigval/Go_Assist/discussions)
- **Bag-reporty:** [GitHub Issues](https://github.com/ezhigval/Go_Assist/issues)
- **Email:** hello@modulr.dev (placeholder)
- **Sayt:** modulr.dev (placeholder)

---

<p align="center">
  <b>Ne pishi prilozheniya. Sobiray ikh.</b><br><br>
  Modulr - infrastruktura dlya tekh, kto tsenit kontrol', privatnost' i gibkost'.
</p>
