# Modulr Frontend

**Cross-platform frontend ecosystem for Modulr personal assistant**

## Overview

Modulr Frontend is a comprehensive React-based application that runs on multiple platforms:
- **Telegram Mini App** - Native integration with Telegram
- **Web/PWA** - Progressive Web App with offline support  
- **Mobile** - iOS and Android via React Native + Capacitor
- **Desktop** - Windows, macOS, Linux via Tauri
- **Wearables** - watchOS and Wear OS (limited UI)

## Current Surface

Текущий реализованный web/PWA слой — это `v2.0 Control Plane`:
- broker lanes с consumer-group конфигурацией и статусами rollout;
- module dispatch matrix для admission / queue mode с inline-редактированием dispatch/group/scope/tag/latency;
- plugin registry для `process|wasm` manifests с operator-редактированием description/capabilities;
- scope presets и quick tags с local persistence fallback;
- единый seed-артефакт `../controlplane/default_snapshot.json` для web fallback и Go-backed operator API;
- `npm run verify` покрывает type-check, vitest и production build.

## Минимальные UX-критерии Web Control Plane

Для `v2.0` web-слоя зафиксирован минимальный UX contract:
- первый экран без прокрутки показывает `health`, `platform`, `active scope`, backend-mode (`memory|persistent|fallback`) и ключевые runtime metrics;
- first-screen snapshot поднимается синхронно из local control-plane state, чтобы operator flow не зависел от network round-trip;
- `/api/health` прокидывает в UI `snapshot freshness`, `persist path`, `plugin manifest source/count`; при недоступности backend UI явно переключается в `local fallback`;
- оператор может вручную перезагрузить plugin manifests через dashboard, если backend доступен и знает `CONTROL_PLANE_PLUGIN_DIR`;
- module cards позволяют править `dispatchMode`, `consumerGroup`, `allowedScopes`, `tags`, `latencyBudgetMs` и сохранять это в snapshot без ручной JSON-правки;
- plugin cards позволяют править `description` и `capabilities` поверх status-cycle, сохраняя operator overrides в backend/local fallback;
- все мутационные действия имеют явные текстовые/ARIA-имена: add scope, rotate broker, toggle module, rotate plugin;
- локальный dev-режим остаётся операбельным без backend: broker/module/plugin/scope изменения пишутся в local snapshot;
- оператор получает мгновенную обратную связь через live event trace (`Control plane booted`, `Config updated`);
- критерии закреплены тестами в `src/test/control-plane.spec.tsx` и входят в `npm run verify`.

## Architecture

### Core Principles
- **LEGO Architecture** - Modular, composable components
- **Event-Driven** - Client-side EventBus synchronized with backend
- **Cross-Platform** - 90%+ code reuse across platforms
- **Offline-First** - Full functionality without internet
- **Type-Safe** - Strict TypeScript throughout

### Technology Stack
- **React 18** - UI framework
- **TypeScript 5** - Type safety
- **Vite 5** - Build tool
- **Tailwind CSS** - Styling
- **Zustand** - State management
- **React Query** - Server state
- **React Native** - Mobile platforms
- **Capacitor** - Native API bridge
- **Tauri** - Desktop platform

## Quick Start

### Prerequisites
- Node.js 18+
- npm 9+
- Git
- Go 1.21+ (если хочешь поднять реальный control-plane backend вместо local fallback)

### Installation

```bash
# Clone the repository
git clone https://github.com/modulr/frontend.git
cd frontend

# Install dependencies
npm install

# Copy environment file
cp .env.example .env.local

# Edit environment variables
nano .env.local
```

### Environment Variables

```bash
# API Configuration
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws

# Platform (auto-detected, but can be overridden)
VITE_PLATFORM=web

# Feature Flags
VITE_ENABLE_ANALYTICS=true
VITE_ENABLE_AI_FEATURES=true

# Development
VITE_DEV_MODE=true
```

### Development

```bash
# Start development server
npm run dev

# Run tests
npm run test

# Type checking
npm run type-check

# Linting
npm run lint

# Format code
npm run format
```

### Real Control Plane Backend

По умолчанию web/PWA слой остаётся операбельным через local snapshot fallback, но для реального operator API можно параллельно запустить Go-projection:

```bash
cd ..
go run ./cmd/controlplane
```

Сервис поднимает:
- `GET /api/health`
- `GET/POST/PATCH/DELETE /api/scopes`
- `GET /api/control-plane`
- `POST /api/control-plane/plugins/reload`
- `PATCH /api/control-plane/modules/:id`
- `PATCH /api/control-plane/plugins/:id`
- `POST /api/control-plane/brokers/:id/cycle`

Дефолтный `VITE_API_BASE_URL` уже указывает на `http://localhost:8080/api`, поэтому фронтенд начнёт использовать этот backend автоматически, а при недоступности сервиса останется на local fallback. Обе стороны стартуют из одного `controlplane/default_snapshot.json`, так что operator-данные не расходятся между fallback и реальным backend. Если backend поднят через `go run ./cmd/controlplane`, его мутации дополнительно сохраняются в `CONTROL_PLANE_STATE_PATH` (по умолчанию `data/controlplane/snapshot.json`), а plugin projection по умолчанию гидратируется из repo manifests `plugins/manifests`; при необходимости источник можно переопределить через `CONTROL_PLANE_PLUGIN_DIR`, а затем перезагрузить manifests прямо из dashboard без рестарта процесса.

На первом экране dashboard оператор видит не только бинарный статус backend, но и режим persistence, свежесть snapshot, путь к file-backed state и источник plugin manifests. Это позволяет отличать реальный backend от `local fallback` без чтения логов.

## Platform-Specific Development

### Telegram Mini App

```bash
# Build for Telegram
npm run build:telegram

# Deploy to Vercel/Netlify
npm run deploy:telegram
```

### Web/PWA

```bash
# Build for web
npm run build:web

# Preview PWA
npm run preview

# Test service worker
npm run test:pwa
```

### Mobile (iOS/Android)

```bash
# Install Capacitor CLI
npm install -g @capacitor/cli

# Add platforms
npx cap add ios
npx cap add android

# Build for mobile
npm run build:mobile

# Sync to native projects
npx cap sync

# Run on iOS
npx cap run ios

# Run on Android
npx cap run android
```

### Desktop (Windows/macOS/Linux)

```bash
# Install Tauri CLI
npm install -g @tauri-apps/cli

# Build for desktop
npm run build:desktop

# Run in development
npm run dev:desktop
```

## Project Structure

```
frontend/
src/
  components/          # Reusable components
    ui/               # Base UI components (Button, Card, etc.)
    layout/           # Layout components (Header, Sidebar)
    modules/          # Business components (Calendar, Tasks)
  context/            # React contexts
    ScopeContext.tsx  # Scope management
    ThemeContext.tsx  # Theme management
  hooks/              # Custom React hooks
    useQuery.ts       # React Query wrapper
    useScope.ts       # Scope management
  lib/                # Utility libraries
    eventBus.ts       # Client-side EventBus
    api.ts            # API client
    utils.ts          # Helper functions
  modules/            # Business logic modules
    calendar/         # Calendar functionality
    tracker/          # Task tracking
    finance/          # Financial management
    control-plane/    # v2.0 broker/module/plugin/scope dashboard
  platforms/          # Platform-specific code
    telegram/         # Telegram Mini App
    web/              # Web/PWA features
    mobile/           # Mobile-specific
    desktop/          # Desktop-specific
  types/              # TypeScript definitions
    core.ts           # Core types
    events.ts         # Event types
    modules.ts        # Module types
    control-plane.ts  # Control plane snapshot types
```

## Key Features

### 1. Scope Management
- Context-aware functionality
- Visual scope indicators
- Quick scope switching
- Persistent scope selection

### 2. Event System
- Real-time updates via WebSocket
- Client-side EventBus
- Offline event queuing
- Event history and replay

### 3. Cross-Platform UI
- Adaptive layouts
- Platform-specific features
- Native integrations
- Consistent design system

### 4. Offline Support
- IndexedDB storage
- Background sync
- Conflict resolution
- Progressive enhancement

### 5. AI Integration
- Smart suggestions
- Natural language processing
- Confidence indicators
- User feedback loops

## Testing

### Unit Tests
```bash
# Run all unit tests
npm run test

# Run with coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

### Integration Tests
```bash
# Run integration tests
npm run test:integration

# API integration tests
npm run test:api
```

### E2E Tests
```bash
# Install Playwright
npm run test:e2e:install

# Run E2E tests
npm run test:e2e

# Run specific platform tests
npm run test:e2e:telegram
npm run test:e2e:web
npm run test:e2e:mobile
```

## Deployment

### Telegram Mini App
1. Build: `npm run build:telegram`
2. Deploy to Vercel/Netlify
3. Configure Telegram Bot with Web App URL
4. Test in Telegram

### Web/PWA
1. Build: `npm run build:web`
2. Deploy to any static hosting
3. Configure service worker
4. Test PWA installation

### Mobile Apps
1. Build: `npm run build:mobile`
2. Sync to native projects: `npx cap sync`
3. Build in Xcode (iOS) or Android Studio
4. Submit to App Store/Google Play

### Desktop Apps
1. Build: `npm run build:desktop`
2. Generate installers: `npm run build:desktop:all`
3. Notarize (macOS) and sign (Windows)
4. Distribute via GitHub Releases

## Performance Optimization

### Bundle Size
- Code splitting by route
- Lazy loading components
- Tree shaking
- Compression

### Runtime Performance
- React.memo for expensive components
- Virtualization for long lists
- Debouncing user input
- Image optimization

### Network Performance
- API response caching
- Image CDN
- Service worker caching
- Prefetching critical resources

## Accessibility

### WCAG 2.1 AA Compliance
- Screen reader support
- Keyboard navigation
- Color contrast
- Focus management

### Features
- High contrast mode
- Reduced motion
- Font size scaling
- Voice control support

## Internationalization

### Supported Languages
- Russian (ru)
- English (en)
- Chinese (zh)
- Spanish (es)

### Implementation
- react-i18next for translations
- RTL language support
- Date/number formatting
- Cultural adaptations

## Security

### Best Practices
- HTTPS everywhere
- Input sanitization
- XSS prevention
- CSRF protection
- Secure token storage

### Data Protection
- No PII in logs
- Encrypted storage
- Secure API communication
- Privacy controls

## Monitoring

### Performance Monitoring
- Core Web Vitals
- Error tracking (Sentry)
- User analytics
- A/B testing

### Health Checks
- API health monitoring
- WebSocket connection status
- Offline sync status
- Feature flag status

## Contributing

### Development Workflow
1. Fork the repository
2. Create feature branch
3. Make changes with tests
4. Run linting and tests
5. Submit pull request

### Code Style
- Follow ESLint rules
- Use Prettier formatting
- Write TypeScript strictly
- Document public APIs

### Commit Messages
- Use conventional commits
- Be descriptive
- Reference issues
- Keep changes focused

## Troubleshooting

### Common Issues

**Build Errors**
```bash
# Clear cache
npm run clean

# Reinstall dependencies
rm -rf node_modules package-lock.json
npm install

# Check TypeScript
npm run type-check
```

**Platform-Specific Issues**

**Telegram:**
- Check WebApp SDK availability
- Verify bot configuration
- Test in Telegram Desktop

**Mobile:**
- Update Capacitor CLI
- Sync native projects
- Check platform permissions

**Desktop:**
- Update Tauri CLI
- Check system dependencies
- Verify build targets

### Getting Help

- [Documentation](./docs/)
- [Issue Tracker](https://github.com/modulr/frontend/issues)
- [Discord Community](https://discord.gg/modulr)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/modulr)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Credits

- [Modulr Team](https://modulr.app)
- [React Community](https://reactjs.org)
- [Vite Contributors](https://vitejs.dev)
- [Tailwind CSS](https://tailwindcss.com)

---

**Built with love for the Modulr ecosystem**
