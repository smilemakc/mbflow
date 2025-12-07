/**
 * Node type schemas aligned with backend implementation
 * Backend reference: internal/domain/workflow/node.go
 */

// Base node configuration interface
export interface BaseNodeConfig {
  [key: string]: any;
}

// HTTP Node Configuration
export interface HTTPNodeConfig extends BaseNodeConfig {
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS";
  headers?: Record<string, string>;
  body?: string;
  timeout_seconds?: number;
  retry_count?: number;
  follow_redirects?: boolean;
}

// Tool Calling Types
export type ToolCallMode = "auto" | "manual";

export type FunctionType =
  | "builtin"
  | "sub_workflow"
  | "custom_code"
  | "openapi";

export interface ToolCallConfig {
  mode: ToolCallMode;
  max_iterations?: number;
  timeout_per_tool?: number;
  total_timeout?: number;
  stop_on_tool_failure?: boolean;
}

export interface FunctionDefinition {
  type: FunctionType;
  name: string;
  description: string;
  parameters?: Record<string, any>; // JSON Schema

  // For FunctionTypeBuiltin
  builtin_name?: string;

  // For FunctionTypeSubWorkflow
  workflow_id?: string;
  input_mapping?: Record<string, string>;
  output_extractor?: string;

  // For FunctionTypeCustomCode
  language?: "javascript" | "python";
  code?: string;

  // For FunctionTypeOpenAPI
  openapi_spec?: string;
  operation_id?: string;
  base_url?: string;
  auth_config?: Record<string, any>;
}

// LLM Node Configuration
export interface LLMNodeConfig extends BaseNodeConfig {
  provider: "openai" | "anthropic" | "google" | "azure" | "ollama";
  model: string;
  api_key: string; // API key (supports templates like {{env.openai_api_key}})

  // Basic settings (always visible)
  instruction?: string; // System prompt (optional)
  prompt: string; // User prompt (required)

  // Advanced settings (progressive disclosure)
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  frequency_penalty?: number;
  presence_penalty?: number;
  stop_sequences?: string[];
  response_format?: "text" | "json";

  // Provider-specific
  timeout_seconds?: number;
  retry_count?: number;

  // Streaming (future)
  stream?: boolean;

  // Tool Calling Configuration (Phase 1)
  tool_call_config?: ToolCallConfig;
  functions?: FunctionDefinition[];
}

// Transform Node Configuration
export interface TransformNodeConfig extends BaseNodeConfig {
  language: "jq" | "javascript";
  expression: string;
  timeout_seconds?: number;
}

// Function Call Node Configuration
export interface FunctionCallNodeConfig extends BaseNodeConfig {
  function_name: string;
  arguments?: Record<string, any>;
  timeout_seconds?: number;
}

// Telegram Node Configuration
export interface TelegramNodeConfig extends BaseNodeConfig {
  bot_token: string;
  chat_id: string;
  message_type: "text" | "photo" | "document" | "audio" | "video";
  text?: string;
  parse_mode?: "Markdown" | "MarkdownV2" | "HTML";
  disable_web_page_preview?: boolean;
  disable_notification?: boolean;
  protect_content?: boolean;

  // Media fields
  file_source?: "base64" | "url" | "file_id";
  file_data?: string;
  file_name?: string;

  timeout_seconds?: number;
}

// File Storage Node Configuration
export interface FileStorageNodeConfig extends BaseNodeConfig {
  action: "store" | "get" | "delete" | "list" | "metadata";
  storage_id?: string;
  file_source?: "url" | "base64";
  file_data?: string;
  file_url?: string;
  file_name?: string;
  mime_type?: string;
  file_id?: string;
  access_scope?: "workflow" | "edge" | "result";
  ttl?: number;
  tags?: string[];
  limit?: number;
  offset?: number;
}

// Conditional Node Configuration
export interface ConditionalNodeConfig extends BaseNodeConfig {
  condition: string; // Expression to evaluate (supports expr-lang)
  true_branch?: string; // Node ID for true branch (optional for UI)
  false_branch?: string; // Node ID for false branch (optional for UI)
}

// Merge Node Configuration
export interface MergeNodeConfig extends BaseNodeConfig {
  merge_strategy?: "first" | "last" | "all" | "custom";
  custom_expression?: string; // Custom merge expression (expr-lang)
}

export type NodeConfig =
  | HTTPNodeConfig
  | LLMNodeConfig
  | TransformNodeConfig
  | FunctionCallNodeConfig
  | TelegramNodeConfig
  | FileStorageNodeConfig
  | ConditionalNodeConfig
  | MergeNodeConfig;

export const NodeType = {
  HTTP: "http",
  LLM: "llm",
  TRANSFORM: "transform",
  FUNCTION_CALL: "function_call",
  TELEGRAM: "telegram",
  FILE_STORAGE: "file_storage",
  CONDITIONAL: "conditional",
  MERGE: "merge",
} as const;

export type NodeType = (typeof NodeType)[keyof typeof NodeType];

// Default configurations for each node type
export const DEFAULT_NODE_CONFIGS: Record<NodeType, NodeConfig> = {
  [NodeType.HTTP]: {
    url: "",
    method: "GET",
    headers: {},
    timeout_seconds: 30,
    retry_count: 0,
    follow_redirects: true,
  },
  [NodeType.LLM]: {
    provider: "openai",
    model: "gpt-4",
    api_key: "",
    prompt: "",
    temperature: 0.7,
    max_tokens: 1000,
  },
  [NodeType.TRANSFORM]: {
    language: "jq",
    expression: ".",
    timeout_seconds: 10,
  },
  [NodeType.FUNCTION_CALL]: {
    function_name: "",
    arguments: {},
    timeout_seconds: 30,
  },
  [NodeType.TELEGRAM]: {
    bot_token: "",
    chat_id: "",
    message_type: "text",
    text: "",
    parse_mode: "HTML",
    timeout_seconds: 30,
  },
  [NodeType.FILE_STORAGE]: {
    action: "store",
    storage_id: "",
    access_scope: "workflow",
    ttl: 0,
  },
  [NodeType.CONDITIONAL]: {
    condition: "{{input.value}} > 0",
  },
  [NodeType.MERGE]: {
    merge_strategy: "all",
  },
};

// LLM Provider models (for dropdown)
export const LLM_PROVIDER_MODELS: Record<string, string[]> = {
  openai: [
    "gpt-4",
    "gpt-4-turbo",
    "gpt-4-turbo-preview",
    "gpt-3.5-turbo",
    "gpt-3.5-turbo-16k",
  ],
  anthropic: [
    "claude-3-opus-20240229",
    "claude-3-sonnet-20240229",
    "claude-3-haiku-20240307",
    "claude-2.1",
    "claude-2.0",
  ],
  google: ["gemini-pro", "gemini-pro-vision"],
  azure: ["gpt-4", "gpt-35-turbo"],
  ollama: ["llama2", "mistral", "codellama"],
};

// HTTP methods for dropdown
export const HTTP_METHODS = [
  "GET",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "HEAD",
  "OPTIONS",
] as const;

// Transform languages for dropdown
export const TRANSFORM_LANGUAGES = ["jq", "javascript"] as const;

// Telegram message types
export const TELEGRAM_MESSAGE_TYPES = [
  "text",
  "photo",
  "document",
  "audio",
  "video",
] as const;
export const TELEGRAM_PARSE_MODES = ["Markdown", "MarkdownV2", "HTML"] as const;
export const TELEGRAM_FILE_SOURCES = ["base64", "url", "file_id"] as const;

// Tool Calling constants
export const TOOL_CALL_MODES: ToolCallMode[] = ["auto", "manual"];

export const FUNCTION_TYPES: FunctionType[] = [
  "builtin",
  "sub_workflow",
  "custom_code",
  "openapi",
];

export const BUILTIN_FUNCTIONS = [
  "get_current_time",
  "get_weather",
  "calculate",
] as const;

export const DEFAULT_TOOL_CALL_CONFIG: ToolCallConfig = {
  mode: "manual",
  max_iterations: 10,
  timeout_per_tool: 30,
  total_timeout: 300,
  stop_on_tool_failure: false,
};

// Node type metadata
export interface NodeTypeMetadata {
  type: NodeType;
  label: string;
  description: string;
  icon: string;
  color: string;
}

export const NODE_TYPE_METADATA: Record<NodeType, NodeTypeMetadata> = {
  [NodeType.HTTP]: {
    type: NodeType.HTTP,
    label: "HTTP Request",
    description: "Make HTTP/HTTPS requests to external APIs",
    icon: "🌐",
    color: "#10B981",
  },
  [NodeType.LLM]: {
    type: NodeType.LLM,
    label: "LLM Call",
    description: "Call language models (OpenAI, Anthropic, etc.)",
    icon: "🤖",
    color: "#8B5CF6",
  },
  [NodeType.TRANSFORM]: {
    type: NodeType.TRANSFORM,
    label: "Transform",
    description: "Transform data using jq or JavaScript",
    icon: "⚡",
    color: "#F59E0B",
  },
  [NodeType.FUNCTION_CALL]: {
    type: NodeType.FUNCTION_CALL,
    label: "Function Call",
    description: "Call custom functions or tools",
    icon: "🔧",
    color: "#3B82F6",
  },
  [NodeType.TELEGRAM]: {
    type: NodeType.TELEGRAM,
    label: "Telegram",
    description: "Send messages via Telegram Bot API",
    icon: "heroicons:paper-airplane",
    color: "#0EA5E9", // Sky blue for Telegram
  },
  [NodeType.FILE_STORAGE]: {
    type: NodeType.FILE_STORAGE,
    label: "File Storage",
    description: "Store, retrieve, and manage files",
    icon: "heroicons:folder",
    color: "#14B8A6", // Teal for Storage
  },
  [NodeType.CONDITIONAL]: {
    type: NodeType.CONDITIONAL,
    label: "Conditional",
    description: "Branch workflow based on conditions",
    icon: "heroicons:code-bracket",
    color: "#EC4899", // Pink for Conditional
  },
  [NodeType.MERGE]: {
    type: NodeType.MERGE,
    label: "Merge",
    description: "Merge results from multiple nodes",
    icon: "heroicons:arrows-pointing-in",
    color: "#A855F7", // Purple for Merge
  },
};
