# Modulr Frontend Rules

**Frontend-ecosystem LEGO-architecture compliant**  
**Version:** 1.0  
**Last updated:** 2025-04-10

---

## 1.0 Architectural Principles

### 1.1 Component Model
- **Presentational Components**: Pure UI, no state logic, receive props/callbacks
- **Container Components**: Business logic, state management, API calls
- **Separation of Concerns**: UI layer never touches backend modules directly
- **Single Responsibility**: Each component has one clear purpose

### 1.2 State Management
- **Zustand**: Global application state (user session, preferences, active scope)
- **React Query**: Server state, caching, synchronization, optimistic updates
- **React Context**: Component tree state (theme, scope context, platform context)
- **Local State**: useState for form inputs, UI toggles, temporary states

### 1.3 Event System
- **Client EventBus**: Typed EventEmitter for frontend events
- **Server Sync**: WebSocket connection for real-time server events
- **Event Types**: Strict TypeScript interfaces, no string literals
- **Event Handlers**: Centralized in hooks, not in components

### 1.4 Modularity
- **Module Isolation**: Each module imports only `@modulr/core-types`
- **No Backend Imports**: Direct imports from `modules/` forbidden
- **API Gateway**: All backend communication through unified API layer
- **Event Boundaries**: Modules communicate only through events

---

## 2.0 Code Style

### 2.1 TypeScript Standards
```typescript
// Strict typing everywhere - no any
interface UserEvent {
  type: 'v1.user.action';
  payload: UserActionPayload;
  timestamp: number;
}

// Functional components only
const Component: React.FC<Props> = ({ data, onAction }) => {
  // Component logic
};

// Proper error handling
const result = await api.call();
if (result.error) {
  logger.error('API call failed', result.error);
  return;
}
```

### 2.2 Component Organization
```
src/components/
  ui/              # Reusable UI primitives
    Button.tsx
    Card.tsx
    Input.tsx
    Modal.tsx
  layout/          # Layout components
    Header.tsx
    Sidebar.tsx
    ContextSwitcher.tsx
    ModuleTabs.tsx
  modules/         # Business components
    calendar/
      CalendarView.tsx
      EventCard.tsx
    tracker/
      TaskList.tsx
      MilestoneCard.tsx
    finance/
      TransactionList.tsx
      BudgetCard.tsx
```

### 2.3 Hook Organization
```
src/hooks/
  useQuery.ts          # React Query wrapper
  useMutation.ts       # Mutation wrapper
  useScope.ts          # Scope context management
  useEventBus.ts       # EventBus subscription
  useOfflineSync.ts    # Offline synchronization
  modules/
    calendar/
      useCalendarEvents.ts
    tracker/
      useTasks.ts
    finance/
      useTransactions.ts
```

### 2.4 Styling Standards
- **Tailwind CSS**: Utility-first approach
- **shadcn/ui**: Component library base
- **CSS Variables**: Theme customization
- **Responsive Design**: Mobile-first approach
- **Accessibility**: WCAG 2.1 AA compliance

---

## 3.0 Platform Adapters

### 3.1 Platform Structure
```
src/platforms/
  telegram/
    TelegramProvider.tsx    # WebApp SDK integration
    TelegramAuth.tsx        # Native authentication
    TelegramTheme.tsx       # Theme adaptation
  web/
    PWAProvider.tsx         # Service Worker integration
    WebPushProvider.tsx     # Push notifications
    WebStorageProvider.tsx  # IndexedDB wrapper
  mobile/
    CapacitorProvider.tsx   # Native API bridge
    BiometricAuth.tsx       # Biometric authentication
    GeolocationProvider.tsx # Location services
  desktop/
    TauriProvider.tsx       # Native desktop features
    SystemTrayProvider.tsx  # System tray integration
    HotkeysProvider.tsx     # Global shortcuts
  wearables/
    WatchProvider.tsx       # Limited UI adaptation
    VoiceInputProvider.tsx  # Voice commands
```

### 3.2 Platform Detection
```typescript
const platform = {
  isTelegram: window.Telegram?.WebApp,
  isMobile: /Android|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent),
  isDesktop: !/Android|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent),
  isWearable: window.navigator.userAgent.includes('Watch')
};
```

---

## 4.0 Backend Integration

### 4.1 API Communication
- **REST Endpoints**: CRUD operations, authentication
- **WebSocket**: Real-time events, live updates
- **GraphQL**: Complex queries (future implementation)
- **Rate Limiting**: Client-side request throttling
- **Retry Logic**: Exponential backoff with jitter

### 4.2 Event Synchronization
```typescript
// Client to Server
clientBus.on('user.action', async (event) => {
  await api.post('/events', event);
});

// Server to Client
ws.onmessage = (event) => {
  const serverEvent = JSON.parse(event.data);
  clientBus.emit(serverEvent.name, serverEvent.payload);
};
```

### 4.3 Offline Strategy
- **IndexedDB**: Local data persistence
- **Offline Queue**: Actions stored locally, synced on reconnect
- **Conflict Resolution**: Timestamp-based with user confirmation
- **Background Sync**: Service Worker for periodic sync

---

## 5.0 Usability Standards

### 5.1 Context Visibility
- **Scope Indicator**: Always visible, color-coded
- **Active Module**: Clear visual hierarchy
- **Navigation Breadcrumbs**: Context trail
- **Quick Switch**: One-click scope changes

### 5.2 Relationship Visualization
- **Connection Lines**: Visual links between related entities
- **Dependency Graph**: Module interconnections
- **Timeline View**: Chronological event relationships
- **Network View**: Entity relationship mapping

### 5.3 AI Integration
- **Suggestion Cards**: Clear AI recommendations
- **Confirmation Buttons**: Explicit user approval required
- **Confidence Indicators**: Visual confidence levels
- **Feedback Loop**: Easy thumbs up/down feedback

### 5.4 Input Optimization
- **Global Quick Add**: Floating action button
- **Voice Input**: Speech-to-text integration
- **Smart Suggestions**: Context-aware autocomplete
- **Keyboard Shortcuts**: Power user features

### 5.5 Adaptive Complexity
- **Progressive Disclosure**: Advanced features hidden by default
- **User Levels**: Beginner, Intermediate, Expert modes
- **Feature Toggles**: Optional advanced functionality
- **Learning Curve**: Guided onboarding

---

## 6.0 Performance Standards

### 6.1 Code Splitting
- **Route-based**: Lazy loading per route
- **Component-based**: Dynamic imports for heavy components
- **Platform-specific**: Conditional loading
- **Feature flags**: On-demand feature loading

### 6.2 Caching Strategy
- **React Query**: Server state caching with invalidation
- **Service Worker**: Static asset caching
- **IndexedDB**: User data and offline content
- **Memory Cache**: Frequently accessed data

### 6.3 Bundle Optimization
- **Tree Shaking**: Remove unused code
- **Minification**: Production builds
- **Compression**: Gzip/Brotli compression
- **CDN Delivery**: Global distribution

---

## 7.0 Security Standards

### 7.1 Data Protection
- **No PII Logging**: Sensitive data never logged
- **Encryption**: All client-server communication encrypted
- **Secure Storage**: Sensitive data in encrypted storage
- **Memory Cleanup**: Sensitive data cleared from memory

### 7.2 Authentication
- **Token Management**: Secure token storage and refresh
- **Session Management**: Proper session handling
- **Multi-factor**: Optional 2FA support
- **Biometric**: Platform-specific biometric auth

### 7.3 Input Validation
- **Type Checking**: Runtime type validation
- **Sanitization**: Input sanitization before processing
- **XSS Prevention**: Proper output encoding
- **CSRF Protection**: Token-based CSRF protection

---

## 8.0 Testing Standards

### 8.1 Unit Tests
- **Component Tests**: React Testing Library
- **Hook Tests**: Custom hook testing
- **Utility Tests**: Pure function testing
- **Coverage**: Minimum 80% coverage

### 8.2 Integration Tests
- **API Tests**: Mock server responses
- **EventBus Tests**: Event flow testing
- **Platform Tests**: Platform-specific features
- **Offline Tests**: Synchronization testing

### 8.3 E2E Tests
- **User Flows**: Critical user journeys
- **Cross-platform**: Multi-platform testing
- **Performance**: Load and performance testing
- **Accessibility**: Screen reader testing

---

## 9.0 Development Standards

### 9.1 Code Quality
- **ESLint**: Strict linting rules
- **Prettier**: Consistent formatting
- **Husky**: Pre-commit hooks
- **TypeScript**: Strict mode enabled

### 9.2 Documentation
- **JSDoc**: Comprehensive API documentation
- **README**: Setup and usage instructions
- **Architecture Docs**: Design decisions documentation
- **Component Docs**: Storybook integration

### 9.3 Version Control
- **Conventional Commits**: Standardized commit messages
- **Branch Strategy**: Feature branch workflow
- **PR Reviews**: Code review requirements
- **CI/CD**: Automated testing and deployment

---

## 10.0 Monitoring and Analytics

### 10.1 Performance Monitoring
- **Core Web Vitals**: LCP, FID, CLS tracking
- **Error Tracking**: Sentry integration
- **Performance Budget**: Bundle size limits
- **Real User Monitoring**: RUM data collection

### 10.2 User Analytics
- **Event Tracking**: User interaction analytics
- **Funnel Analysis**: Conversion tracking
- **A/B Testing**: Feature experimentation
- **Heat Maps**: User behavior visualization

---

## 11.0 Accessibility Standards

### 11.1 WCAG Compliance
- **Level AA**: WCAG 2.1 AA compliance
- **Screen Readers**: NVDA, VoiceOver support
- **Keyboard Navigation**: Full keyboard accessibility
- **Color Contrast**: 4.5:1 contrast ratio minimum

### 11.2 Assistive Technology
- **ARIA Labels**: Proper ARIA attributes
- **Semantic HTML**: Correct HTML5 semantics
- **Focus Management**: Logical focus flow
- **Alternative Text**: Image descriptions

---

## 12.0 Internationalization

### 12.1 Multi-language Support
- **i18n Framework**: react-i18next integration
- **Localization**: Date, number, currency formatting
- **RTL Support**: Right-to-left language support
- **Dynamic Loading**: Language pack lazy loading

### 12.2 Cultural Adaptation
- **Color Psychology**: Culturally appropriate colors
- **Iconography**: Culturally neutral icons
- **Text Direction**: Proper text direction handling
- **Content Adaptation**: Region-specific content

---

**Remember: Frontend is a display layer. All business logic lives in the backend.**
