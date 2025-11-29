# Integration Tests with Backend

Комплексные интеграционные тесты для проверки взаимодействия фронтенда с REST API бэкенда MBFlow.

## 📋 Содержание

- [Обзор](#обзор)
- [Требования](#требования)
- [Запуск тестов](#запуск-тестов)
- [Структура тестов](#структура-тестов)
- [Покрытие](#покрытие)
- [Troubleshooting](#troubleshooting)

## Обзор

Интеграционные тесты проверяют:

1. **Workflows API** - CRUD операции с workflows, nodes, edges
2. **Executions API** - Выполнение workflows, отслеживание статуса, события
3. **E2E Scenarios** - Полные сценарии от создания до выполнения workflows

## Требования

### Backend Server

Для запуска интеграционных тестов необходим запущенный бэкенд сервер:

```bash
# Из корня проекта
cd /Users/balashov/PycharmProjects/mbflow
go run cmd/server/main.go
```

Сервер должен быть доступен по адресу: `http://localhost:8181`

### Проверка доступности

```bash
# Проверить health endpoint
curl http://localhost:8181/health

# Должен вернуть: {"status":"ok"}
```

## Запуск тестов

### Все интеграционные тесты

```bash
# Из директории mbflow-ui
npm run test -- src/tests/integration
```

### Конкретный файл тестов

```bash
# Workflows API
npm run test -- src/tests/integration/api/workflows.api.spec.ts

# Executions API
npm run test -- src/tests/integration/api/executions.api.spec.ts

# E2E сценарии
npm run test -- src/tests/integration/api/e2e-scenarios.spec.ts
```

### С UI интерфейсом

```bash
npm run test:ui
```

Затем выберите интеграционные тесты в UI.

### В режиме watch

```bash
npm run test -- src/tests/integration --watch
```

### С покрытием

```bash
npm run test:coverage -- src/tests/integration
```

## Структура тестов

```text
src/tests/integration/
├── api/
│   ├── workflows.api.spec.ts      # Тесты Workflows API
│   ├── executions.api.spec.ts     # Тесты Executions API
│   └── e2e-scenarios.spec.ts      # E2E сценарии
├── helpers/
│   └── test-utils.ts              # Утилиты для тестов
├── config.ts                       # Конфигурация тестов
└── README.md                       # Эта документация
```

## Покрытие

### Workflows API Tests (`workflows.api.spec.ts`)

**Workflow CRUD:**

- ✅ Создание workflow
- ✅ Получение списка workflows
- ✅ Получение workflow по ID
- ✅ Обновление workflow
- ✅ Удаление workflow
- ✅ Получение графа workflow

**Node Operations:**

- ✅ Список nodes в workflow
- ✅ Создание node
- ✅ Получение node по ID
- ✅ Обновление node
- ✅ Удаление node
- ✅ Получение доступных типов nodes

**Edge Operations:**

- ✅ Список edges в workflow
- ✅ Создание edge
- ✅ Получение edge по ID
- ✅ Создание conditional edge
- ✅ Обновление edge
- ✅ Удаление edge
- ✅ Получение доступных типов edges

### Executions API Tests (`executions.api.spec.ts`)

**Execution Operations:**

- ✅ Выполнение workflow
- ✅ Получение списка executions
- ✅ Фильтрация по workflow ID
- ✅ Фильтрация по статусу
- ✅ Получение execution по ID
- ✅ Получение событий execution
- ✅ Выполнение с комплексными переменными

**Lifecycle:**

- ✅ Отслеживание жизненного цикла execution
- ✅ События жизненного цикла
- ✅ Параллельное выполнение workflows

**Error Handling:**

- ✅ Обработка невалидного workflow ID
- ✅ Обработка невалидного execution ID
- ✅ Обработка отсутствующих переменных

### E2E Scenarios (`e2e-scenarios.spec.ts`)

**Complete Workflows:**

- ✅ Simple Transform Workflow
- ✅ Conditional Routing Workflow
- ✅ HTTP Integration Workflow
- ✅ Workflow Modification and Re-execution
- ✅ Multi-Step Data Pipeline
- ✅ Error Recovery

## Утилиты

### Test Utils (`helpers/test-utils.ts`)

```typescript
import {
    isBackendAvailable,
    waitForBackend,
    waitForExecutionCompletion,
    cleanupWorkflows,
    generateTestName,
    retryWithBackoff,
    assertExecutionSuccess,
    assertWorkflowValid,
    createSimpleWorkflowData,
    PerformanceTimer,
    TestDataGenerators,
} from './helpers/test-utils'
```

**Основные функции:**

- `isBackendAvailable()` - Проверка доступности бэкенда
- `waitForExecutionCompletion()` - Ожидание завершения execution
- `cleanupWorkflows()` - Очистка тестовых workflows
- `assertExecutionSuccess()` - Проверка успешного выполнения
- `assertWorkflowValid()` - Валидация структуры workflow

## Конфигурация

Настройки в `config.ts`:

```typescript
export const INTEGRATION_TEST_CONFIG = {
    apiBaseUrl: 'http://localhost:8181',
    apiTimeout: 30000,
    executionTimeout: 60000,
    executionPollInterval: 500,
    maxRetries: 3,
    cleanupAfterTests: true,
}
```

### Переменные окружения

```bash
# API base URL
export VITE_API_BASE_URL=http://localhost:8181

# Verbose logging
export VITEST_VERBOSE=true

# Skip tests if backend unavailable
export SKIP_INTEGRATION_TESTS=false
```

## Troubleshooting

### Backend не запущен

**Ошибка:**

```
Error: Backend server is not available
```

**Решение:**

```bash
cd /Users/balashov/PycharmProjects/mbflow
go run cmd/server/main.go
```

### Timeout ошибки

**Ошибка:**

```
Error: Execution did not complete within 30000ms
```

**Решение:**
Увеличьте timeout в `config.ts`:

```typescript
executionTimeout: 120000, // 2 минуты
```

### Конфликты портов

**Ошибка:**

```
Error: connect ECONNREFUSED 127.0.0.1:8181
```

**Решение:**
Проверьте, что порт 8181 свободен:

```bash
lsof -i :8181
```

### Тесты падают случайно

**Решение:**
Используйте retry механизм:

```typescript
import { retryWithBackoff } from './helpers/test-utils'

await retryWithBackoff(async () => {
    return await api.someOperation()
}, 3, 1000)
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      mbflow-backend:
        image: mbflow-api:latest
        ports:
          - 8181:8181
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: npm ci
        working-directory: ./mbflow-ui
      
      - name: Wait for backend
        run: |
          timeout 30 bash -c 'until curl -f http://localhost:8181/health; do sleep 1; done'
      
      - name: Run integration tests
        run: npm run test -- src/tests/integration
        working-directory: ./mbflow-ui
```

## Best Practices

1. **Cleanup**: Всегда очищайте тестовые данные в `afterAll`
2. **Isolation**: Каждый тест должен быть независимым
3. **Timeouts**: Используйте адекватные timeouts для async операций
4. **Assertions**: Проверяйте не только успех, но и структуру данных
5. **Error Handling**: Тестируйте как success, так и error сценарии
6. **Naming**: Используйте уникальные имена для тестовых workflows

## Связанные документы

- [API Documentation](../../../API_README.md)
- [Unit Tests](../unit/README.md)
- [E2E Tests](../e2e/README.md)

## Поддержка

При возникновении проблем:

1. Проверьте, что бэкенд запущен и доступен
2. Проверьте логи бэкенда: `go run cmd/server/main.go`
3. Запустите тесты с verbose флагом: ```bash
VITEST_VERBOSE=true npm run test

```
4. Создайте issue с логами и описанием проблемы
