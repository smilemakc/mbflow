import { useUIStore } from './uiStore';

const dictionaries = {
  en: {
    common: {
      save: "Save",
      run: "Run",
      running: "Running...",
      undo: "Undo",
      redo: "Redo",
      saved: "Saved",
      unsaved: "Unsaved",
      saving: "Saving...",
      lastSaved: "Last saved",
      delete: "Delete",
      cancel: "Cancel",
      search: "Search...",
      loading: "Loading",
      error: "Error",
      success: "Success",
      unknown: "Unknown"
    },
    sidebar: {
      dashboard: "Dashboard",
      workflows: "Workflows",
      executions: "Executions",
      triggers: "Triggers",
      monitoring: "Monitoring",
      resources: "Resources",
      settings: "Settings",
      signOut: "Sign Out",
      title: "Workflow.ai"
    },
    dashboard: {
      title: "Dashboard Overview",
      subtitle: "Monitor your automation health and performance.",
      totalWorkflows: "Total Workflows",
      executionsToday: "Executions Today",
      successRate: "Success Rate",
      avgDuration: "Avg Duration",
      recentActivity: "Recent Activity",
      viewAll: "View All",
      systemHealth: "System Health",
      quickActions: "Ready to build?",
      createWorkflow: "Create New Workflow",
      quickDesc: "Create a new workflow using the visual editor.",
      table: {
        workflow: "Workflow",
        status: "Status",
        duration: "Duration",
        triggeredBy: "Triggered By",
        time: "Time"
      }
    },
    builder: {
      components: "Components",
      dragToAdd: "Drag to add",
      triggers: "TRIGGERS",
      actions: "ACTIONS",
      telegram: "TELEGRAM",
      logic: "LOGIC",
      storage: "STORAGE",
      adapters: "ADAPTERS",
      properties: "Properties",
      general: "General",
      name: "Name",
      description: "Description",
      deleteNode: "Delete Node",
      nodeConfig: "Config"
    },
    monitoring: {
      title: "Execution Monitor",
      consoleLogs: "Console Logs",
      io: "Inputs / Outputs",
      clear: "Clear Logs",
      ready: "Ready to execute. Click 'Run' to start.",
      inputs: "INPUTS",
      outputs: "OUTPUTS",
      selectNode: "Select a node to view its inputs and outputs."
    },
    settings: {
      title: "Settings",
      profile: "Profile",
      notifications: "Notifications",
      security: "Security",
      billing: "Billing",
      appearance: "Appearance",
      profileInfo: "Profile Information",
      profileDesc: "Update your account details and email.",
      firstName: "First Name",
      lastName: "Last Name",
      email: "Email Address",
      bio: "Bio",
      saveChanges: "Save Changes"
    },
    nodes: {
      // Triggers
      delay: "Delay / Scheduler",

      // Actions
      http: "HTTP Request",
      llm: "LLM / AI",
      transform: "Transform",
      functionCall: "Function Call",

      // Telegram
      telegram: "Telegram Send",
      telegramDownload: "TG Download",
      telegramParse: "TG Parse",
      telegramCallback: "TG Callback",

      // Logic
      conditional: "Conditional",
      merge: "Merge",

      // Storage
      fileStorage: "File Storage",

      // Adapters
      HTMLCleaner: "HTML cleaner",
      base64ToBytes: "Base64 → Bytes",
      bytesToBase64: "Bytes → Base64",
      stringToJson: "String → JSON",
      jsonToString: "JSON → String",
      bytesToJson: "Bytes → JSON",
      fileToBytes: "File → Bytes",
      bytesToFile: "Bytes → File",
      csvToJson: "CSV → JSON",

      // External integrations
      googleSheets: "Google Sheets",
      googleDrive: "Google Drive",

      // Legacy keys (for backwards compatibility)
      scheduler: "Scheduler",
      telegramBot: "Telegram Bot",
      apiRequest: "HTTP Request",
      aiGenerator: "AI Generator",
      condition: "Condition"
    },
    executions: {
      title: "Execution History",
      subtitle: "View and manage all workflow executions",
      filters: "Filters",
      allWorkflows: "All Workflows",
      allStatuses: "All Statuses",
      dateRange: "Date Range",
      applyFilters: "Apply Filters",
      clearFilters: "Clear Filters",
      table: {
        id: "ID",
        workflow: "Workflow",
        status: "Status",
        startedAt: "Started At",
        duration: "Duration",
        triggeredBy: "Triggered By",
        actions: "Actions"
      },
      status: {
        pending: "Pending",
        running: "Running",
        completed: "Completed",
        failed: "Failed",
        cancelled: "Cancelled"
      },
      actions: {
        viewDetails: "View Details",
        retry: "Retry",
        cancel: "Cancel"
      },
      details: {
        title: "Execution Details",
        overview: "Overview",
        nodeExecutions: "Node Executions",
        logs: "Logs",
        input: "Input",
        output: "Output",
        error: "Error",
        close: "Close"
      },
      noData: "No executions found",
      loadMore: "Load More",
      pagination: {
        showing: "Showing",
        of: "of",
        results: "results"
      }
    },
    executionDetail: {
      title: "Execution Details",
      backToList: "Back to Executions",
      refresh: "Refresh",
      copy: "Copy",
      copied: "Copied!",
      noData: "No data",
      fields: "fields",
      error: "Error",
      input: "Input",
      output: "Output",
      startedAt: "Started At",
      completedAt: "Completed At",
      retryCount: "Retry Count",
      totalNodes: "Total Nodes",
      nodeExecutions: "Node Executions",
      nodeExecutionsDesc: "Click on a node to view its inputs and outputs",
      expandAll: "Expand All",
      collapseAll: "Collapse All",
      noNodeExecutions: "No node executions found",
      overview: "Overview",
      metadata: "Metadata",
      variables: "Variables",
      executionError: "Execution Error",
      notFound: "Execution not found",
      fetchError: "Failed to load execution details",
      retryStarted: "Retry started",
      retryFailed: "Failed to retry execution"
    },
    auth: {
      signIn: "Sign In",
      signUp: "Sign Up",
      signOut: "Sign Out",
      signInDescription: "Enter your credentials to access your account",
      signUpDescription: "Create a new account to get started",
      signingIn: "Signing in...",
      creatingAccount: "Creating account...",
      createAccount: "Create Account",
      email: "Email",
      password: "Password",
      confirmPassword: "Confirm Password",
      username: "Username",
      fullName: "Full Name",
      noAccount: "Don't have an account?",
      alreadyHaveAccount: "Already have an account?",
      passwordsDoNotMatch: "Passwords do not match",
      passwordTooShort: "Password must be at least 8 characters",
      usernameTooShort: "Username must be at least 3 characters",
      usernameHint: "Letters, numbers, underscores and hyphens only",
      passwordHint: "At least 8 characters with uppercase, lowercase, and number"
    },
    nav: {
      settings: "Settings",
      userManagement: "User Management"
    },
    admin: {
      userManagement: "User Management",
      userManagementDescription: "Manage users and their roles",
      addUser: "Add User",
      user: "User",
      email: "Email",
      roles: "Roles",
      status: "Status",
      actions: "Actions"
    }
  },
  ru: {
    common: {
      save: "Сохранить",
      run: "Запуск",
      running: "Выполнение...",
      undo: "Отменить",
      redo: "Вернуть",
      saved: "Сохранено",
      unsaved: "Не сохранено",
      saving: "Сохранение...",
      lastSaved: "Сохр.",
      delete: "Удалить",
      cancel: "Отмена",
      search: "Поиск...",
      loading: "Загрузка",
      error: "Ошибка",
      success: "Успех",
      unknown: "Неизвестно"
    },
    sidebar: {
      dashboard: "Дашборд",
      workflows: "Процессы",
      executions: "История",
      triggers: "Триггеры",
      monitoring: "Мониторинг",
      resources: "Ресурсы",
      settings: "Настройки",
      signOut: "Выйти",
      title: "Workflow.ai"
    },
    dashboard: {
      title: "Обзор системы",
      subtitle: "Мониторинг здоровья и производительности автоматизаций.",
      totalWorkflows: "Всего процессов",
      executionsToday: "Запусков сегодня",
      successRate: "Успешность",
      avgDuration: "Ср. время",
      recentActivity: "Недавняя активность",
      viewAll: "Показать все",
      systemHealth: "Здоровье системы",
      quickActions: "Готовы создать?",
      createWorkflow: "Создать процесс",
      quickDesc: "Создайте новый процесс в визуальном редакторе.",
      table: {
        workflow: "Процесс",
        status: "Статус",
        duration: "Время",
        triggeredBy: "Источник",
        time: "Когда"
      }
    },
    builder: {
      components: "Компоненты",
      dragToAdd: "Перетащите",
      triggers: "ТРИГГЕРЫ",
      actions: "ДЕЙСТВИЯ",
      telegram: "TELEGRAM",
      logic: "ЛОГИКА",
      storage: "ХРАНИЛИЩЕ",
      adapters: "АДАПТЕРЫ",
      properties: "Свойства",
      general: "Основное",
      name: "Название",
      description: "Описание",
      deleteNode: "Удалить узел",
      nodeConfig: "Конфигурация"
    },
    monitoring: {
      title: "Монитор выполнения",
      consoleLogs: "Логи консоли",
      io: "Входы / Выходы",
      clear: "Очистить",
      ready: "Готов к запуску. Нажмите 'Запуск'.",
      inputs: "ВХОДНЫЕ ДАННЫЕ",
      outputs: "ВЫХОДНЫЕ ДАННЫЕ",
      selectNode: "Выберите узел для просмотра данных."
    },
    settings: {
      title: "Настройки",
      profile: "Профиль",
      notifications: "Уведомления",
      security: "Безопасность",
      billing: "Оплата",
      appearance: "Внешний вид",
      profileInfo: "Информация профиля",
      profileDesc: "Обновите данные аккаунта и email.",
      firstName: "Имя",
      lastName: "Фамилия",
      email: "Email адрес",
      bio: "О себе",
      saveChanges: "Сохранить изменения"
    },
    nodes: {
      // Triggers
      delay: "Задержка / Планировщик",

      // Actions
      http: "HTTP Запрос",
      llm: "LLM / AI",
      transform: "Трансформация",
      functionCall: "Вызов функции",

      // Telegram
      telegram: "Telegram отправка",
      telegramDownload: "TG Загрузка",
      telegramParse: "TG Парсинг",
      telegramCallback: "TG Callback",

      // Logic
      conditional: "Условие",
      merge: "Объединение",

      // Storage
      fileStorage: "Файловое хранилище",

      // Adapters
      HTMLCleaner: "HTML клинер",
      base64ToBytes: "Base64 → Байты",
      bytesToBase64: "Байты → Base64",
      stringToJson: "Строка → JSON",
      jsonToString: "JSON → Строка",
      bytesToJson: "Байты → JSON",
      fileToBytes: "Файл → Байты",
      bytesToFile: "Байты → Файл",
      csvToJson: "CSV → JSON",

      // External integrations
      googleSheets: "Google Таблицы",
      googleDrive: "Google Диск",

      // Legacy keys (for backwards compatibility)
      scheduler: "Планировщик",
      telegramBot: "Telegram Бот",
      apiRequest: "HTTP Запрос",
      aiGenerator: "AI Генератор",
      condition: "Условие"
    },
    executions: {
      title: "История выполнений",
      subtitle: "Просмотр и управление выполнениями процессов",
      filters: "Фильтры",
      allWorkflows: "Все процессы",
      allStatuses: "Все статусы",
      dateRange: "Период",
      applyFilters: "Применить",
      clearFilters: "Сбросить",
      table: {
        id: "ID",
        workflow: "Процесс",
        status: "Статус",
        startedAt: "Запущен",
        duration: "Время",
        triggeredBy: "Источник",
        actions: "Действия"
      },
      status: {
        pending: "Ожидание",
        running: "Выполняется",
        completed: "Завершен",
        failed: "Ошибка",
        cancelled: "Отменен"
      },
      actions: {
        viewDetails: "Детали",
        retry: "Повтор",
        cancel: "Отмена"
      },
      details: {
        title: "Детали выполнения",
        overview: "Обзор",
        nodeExecutions: "Выполнение узлов",
        logs: "Логи",
        input: "Вход",
        output: "Выход",
        error: "Ошибка",
        close: "Закрыть"
      },
      noData: "Выполнений не найдено",
      loadMore: "Загрузить еще",
      pagination: {
        showing: "Показано",
        of: "из",
        results: "результатов"
      }
    },
    executionDetail: {
      title: "Детали выполнения",
      backToList: "К списку выполнений",
      refresh: "Обновить",
      copy: "Копировать",
      copied: "Скопировано!",
      noData: "Нет данных",
      fields: "полей",
      error: "Ошибка",
      input: "Входные данные",
      output: "Выходные данные",
      startedAt: "Запущен",
      completedAt: "Завершен",
      retryCount: "Попыток",
      totalNodes: "Всего узлов",
      nodeExecutions: "Выполнение узлов",
      nodeExecutionsDesc: "Нажмите на узел для просмотра входов и выходов",
      expandAll: "Развернуть все",
      collapseAll: "Свернуть все",
      noNodeExecutions: "Выполнения узлов не найдены",
      overview: "Обзор",
      metadata: "Метаданные",
      variables: "Переменные",
      executionError: "Ошибка выполнения",
      notFound: "Выполнение не найдено",
      fetchError: "Не удалось загрузить детали выполнения",
      retryStarted: "Повторный запуск начат",
      retryFailed: "Не удалось повторить выполнение"
    },
    auth: {
      signIn: "Войти",
      signUp: "Регистрация",
      signOut: "Выйти",
      signInDescription: "Введите свои данные для входа в аккаунт",
      signUpDescription: "Создайте новый аккаунт",
      signingIn: "Вход...",
      creatingAccount: "Создание аккаунта...",
      createAccount: "Создать аккаунт",
      email: "Email",
      password: "Пароль",
      confirmPassword: "Подтвердите пароль",
      username: "Имя пользователя",
      fullName: "Полное имя",
      noAccount: "Нет аккаунта?",
      alreadyHaveAccount: "Уже есть аккаунт?",
      passwordsDoNotMatch: "Пароли не совпадают",
      passwordTooShort: "Пароль должен быть не менее 8 символов",
      usernameTooShort: "Имя пользователя должно быть не менее 3 символов",
      usernameHint: "Только буквы, цифры, подчеркивания и дефисы",
      passwordHint: "Минимум 8 символов с заглавной, строчной буквой и цифрой"
    },
    nav: {
      settings: "Настройки",
      userManagement: "Управление пользователями"
    },
    admin: {
      userManagement: "Управление пользователями",
      userManagementDescription: "Управление пользователями и их ролями",
      addUser: "Добавить пользователя",
      user: "Пользователь",
      email: "Email",
      roles: "Роли",
      status: "Статус",
      actions: "Действия"
    }
  }
};

export const useTranslation = () => {
  const { language } = useUIStore();

  // Return the dictionary for the current language
  // Fallback to English if something goes wrong, though types prevent valid keys
  return dictionaries[language] || dictionaries.en;
};

// Helper function for translations with fallback support
// Usage: t('auth.signIn', 'Sign In') or t('auth.signIn')
export const useTranslations = () => {
  const { language } = useUIStore();
  const dict = dictionaries[language] || dictionaries.en;
  const fallbackDict = dictionaries.en;

  return (key: string, fallback?: string): string => {
    const keys = key.split('.');
    let value: unknown = dict;
    let fallbackValue: unknown = fallbackDict;

    for (const k of keys) {
      if (value && typeof value === 'object' && k in value) {
        value = (value as Record<string, unknown>)[k];
      } else {
        value = undefined;
      }
      if (fallbackValue && typeof fallbackValue === 'object' && k in fallbackValue) {
        fallbackValue = (fallbackValue as Record<string, unknown>)[k];
      } else {
        fallbackValue = undefined;
      }
    }

    if (typeof value === 'string') {
      return value;
    }
    if (typeof fallbackValue === 'string') {
      return fallbackValue;
    }
    return fallback ?? key;
  };
};