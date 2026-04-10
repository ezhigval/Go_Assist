# 🤝 Руководство по контрибуции в Modulr

Спасибо, что хочешь внести вклад в развитие проекта! 🧱  
Modulr — это open-source платформа, и мы ценим любую помощь: код, документацию, тесты, идеи, баг-репорты.

## 📜 Кодекс поведения

Мы придерживаемся принципов уважительного общения.  
Недопустимы: оскорбления, дискриминация, спам, агрессивный тон.  
Нарушения могут привести к бану в репозитории и сообществе.

## 🚀 С чего начать

1. **Изучи документацию**:
   - [README.md](./README.md) — обзор проекта
   - [PROJECT_RULES.md](./PROJECT_RULES.md) — архитектурные стандарты
   - [ROADMAP.md](./ROADMAP.md) — что в работе, что планируется

2. **Найди задачу**:
   - [`good first issue`](https://github.com/ezhigval/Go_Assist/issues?q=label%3A%22good+first+issue%22) — для новичков
   - [`help wanted`](https://github.com/ezhigval/Go_Assist/issues?q=label%3A%22help+wanted%22) — нужна помощь сообщества
   - [`bug`](https://github.com/ezhigval/Go_Assist/issues?q=label%3Abug) — исправление ошибок

3. **Обсуди идею**:
   - Для новых фич создай [Discussion](https://github.com/ezhigval/Go_Assist/discussions) или напиши @ezhigval в Telegram
   - Убедись, что задача не дублирует существующую

## 🛠️ Процесс контрибуции

### 1. Форк и клонирование
```bash
# Форкни репозиторий через GitHub UI
git clone https://github.com/ТВОЙ_ЮЗЕР/Go_Assist.git
cd Go_Assist
git remote add upstream https://github.com/ezhigval/Go_Assist.git

Создай ветку
```bash
git checkout -b feature/твоя-фича
# Или: fix/багфикс, docs/обновление, refactor/улучшение
```

### 3. Разрабатывай
Следуй PROJECT_RULES.md
Пиши тесты для новой логики
Комментируй публичные методы на русском
Не ломай существующие контракты

### 4. Проверь код
```bash
# Go
cd core && go fmt ./... && go vet ./... && go test ./...
```

# Frontend
cd frontend && npm run lint && npm run test

# Python/AI
cd ai && black . && mypy . && pytest .

# Общий линтинг (если есть)
make lint  # или npm run lint:all

### 5. Закоммить и запушь
```bash
git add .
git commit -m "feat(module): краткое описание изменений

- что сделано
- почему это важно
- связанные issues: #123"

git push origin feature/твоя-фича
```

6. Создай Pull Request
Заполни шаблон PR (если есть)
Опиши изменения, скриншоты/видео для UI
Укажи связанные issues: Closes #123
Дождись ревью и правок
## 📐 Стандарты кода
### Go
- go fmt, go vet, staticcheck — обязательно
- context.Context первым параметром во всех функциях
- Обработка ошибок: fmt.Errorf("...: %w", err)
- Никаких глобальных переменных, только интерфейсы

### Frontend (React/TypeScript)
- Strict mode: noImplicitAny, strictNullChecks
- Функциональные компоненты + хуки
- Tailwind CSS для стилей, shadcn/ui для компонентов
- Тесты: Vitest + React Testing Library
### Python (AI/ML)
- Type hints: def func(x: int) -> str:
- Форматирование: black, isort
- Линтинг: flake8, mypy
- Тесты: pytest, мокай внешние зависимости

### Общие
- Комментарии на русском, код/символы на английском
- Именование: camelCase для экспорта, PascalCase для типов, snake_case для БД/событий
- Никаких console.log / fmt.Println в продакшен-коде (используй логгер)

## 🧪 Тестирование
Минимальные требования для PR:
Новые функции покрыты юнит-тестами
Интеграционные тесты для EventBus/API
E2E-тест для критичных пользовательских сценариев
Все тесты проходят: make test или npm run test:all
Запуск тестов:

```bash
# Все тесты
make test

# Только юнит-тесты
make test:unit

# Только интеграционные
make test:integration

# С покрытием
make test:coverage
# Отчёт: coverage.html
```

## 📝 Документация
- Обновляй README, если меняешь публичное API
- Добавляй примеры использования для новых модулей
- Комментируй сложные алгоритмы и архитектурные решения
- Используй docs/ для детальных гайдов
## 🤖 AI-контрибуции
Если ты добавляешь новую модель или AI-фичу:
- Опиши задачу и метрики успеха в ai/
- Добавь JSON Schema для Input/Output
- Реализуй fallback на правила при недоступности модели
- Протестируй на реальных данных (анонимизированных)
- Обнови AI_RULES.md
## 🏷️ Тегирование коммитов
Используй конвенциональные коммиты:

```bash
feat(module): добавить новую фичу
fix(module): исправить баг
docs: обновить документацию
style: форматирование, без изменения логики
refactor: улучшение кода без изменения поведения
test: добавить или исправить тесты
chore: обновление зависимостей, конфигов, инструментов
```

## 🎁 Награды за контрибуции
- 🌟 Имя в списке контрибьюторов в README
- 🏷️ Бейдж "Contributor" в сообществе
- 🎨 Эксклюзивный мерч (при значимом вкладе)
- 💰 Оплата за сложные задачи (через спонсорство)
- 🧠 Менторство и код-ревью от мейнтейнеров
## ❓ Вопросы?
- 💬 GitHub Discussions
- 💬 Telegram: @ezhigval
- 🐛 Баг-репорты

🧱 Каждый кирпичик важен. Спасибо, что строишь Modulr вместе с нами!

