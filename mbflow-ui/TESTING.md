# MBFlow UI - Testing Guide

Полноценная тестовая инфраструктура для фронтенда MBFlow.

## Типы тестов

### 1. Unit тесты (Vitest)

Тестируют отдельные утилиты и функции.

**Запуск:**

```bash
# Запустить все unit тесты
npm run test

# Запустить с UI
npm run test:ui

# Запустить один раз
npm run test:run

# С покрытием кода
npm run test:coverage
```

**Примеры тестов:**

- `src/tests/unit/formatting.spec.ts` - тесты для `toSnakeCase` и `toTitleCase`
- `src/tests/unit/name-generator.spec.ts` - тесты для генератора случайных имен

### 2. Integration тесты (Vitest)

Тестируют взаимодействие компонентов и stores.

**Примеры:**

- `src/tests/integration/workflow-store.spec.ts` - тесты для Workflow Store

### 3. E2E тесты (Playwright)

Тестируют полный пользовательский сценарий в браузере.

**Запуск:**

```bash
# Запустить E2E тесты
npm run test:e2e

# С UI интерфейсом
npm run test:e2e:ui

# В режиме отладки
npm run test:e2e:debug
```

**Примеры тестов:**

- `src/tests/e2e/workflow-editor.spec.ts` - полный цикл создания и редактирования workflow

## Структура тестов

```
src/tests/
├── setup.ts                          # Настройка тестового окружения
├── unit/                             # Unit тесты
│   ├── formatting.spec.ts
│   └── name-generator.spec.ts
├── integration/                      # Integration тесты
│   └── workflow-store.spec.ts
└── e2e/                              # E2E тесты
    └── workflow-editor.spec.ts
```

## Покрываемая функциональность

### ✅ Unit тесты

- [x] Форматирование имен (snake_case ↔ Title Case)
- [x] Генератор случайных имен
- [x] Уникальность сгенерированных имен
- [x] Валидация форматов

### ✅ Integration тесты

- [x] Создание workflow
- [x] Добавление/удаление нод
- [x] Создание/удаление связей
- [x] Генерация уникальных имен для нод
- [x] Валидация данных workflow

### ✅ E2E тесты

- [x] Загрузка редактора workflow
- [x] Отображение палитры нод
- [x] Drag & Drop нод на canvas
- [x] Создание связей между нодами
- [x] Автоматическое форматирование имен
- [x] Сохранение workflow
- [x] Удаление нод

## Результаты тестирования

**Последний запуск:**

- ✅ 16 unit тестов прошли успешно
- ✅ Все тесты форматирования работают корректно
- ✅ Генератор имен проходит все проверки

## Конфигурация

### Vitest

Конфигурация: `vitest.config.ts`

- Окружение: `happy-dom`
- Покрытие: v8 provider
- Глобальные моки для browser APIs

### Playwright

Конфигурация: `playwright.config.ts`

- Браузер: Chromium
- Base URL: `http://localhost:3434`
- Автозапуск dev сервера

## Запуск всех тестов

```bash
# Unit + Integration тесты
npm run test:run

# E2E тесты
npm run test:e2e

# Все вместе
npm run test:run && npm run test:e2e
```

## CI/CD Integration

Тесты готовы для интеграции в CI/CD pipeline:

- Vitest поддерживает JUnit reporter
- Playwright генерирует HTML отчеты
- Покрытие кода экспортируется в JSON/HTML

## Отладка

### Unit/Integration тесты

```bash
# Запустить с UI для интерактивной отладки
npm run test:ui
```

### E2E тесты

```bash
# Запустить в режиме отладки
npm run test:e2e:debug

# Или с UI
npm run test:e2e:ui
```

## Добавление новых тестов

### Unit тест

```typescript
import { describe, it, expect } from 'vitest'

describe('My Feature', () => {
  it('should work correctly', () => {
    expect(myFunction()).toBe(expected)
  })
})
```

### E2E тест

```typescript
import { test, expect } from '@playwright/test'

test('should perform action', async ({ page }) => {
  await page.goto('/path')
  await expect(page.locator('.element')).toBeVisible()
})
```

## Известные проблемы

- ⚠️ Конфликт версий Vite между Vitest и основным проектом (не критично)
- ⚠️ E2E тесты не должны запускаться через `npm run test` (только через `test:e2e`)

## Метрики качества

- **Покрытие кода**: Цель >80%
- **Время выполнения unit тестов**: <1s
- **Время выполнения E2E тестов**: <30s
