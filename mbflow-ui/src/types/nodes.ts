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

// Union type for all node configs
export type NodeConfig =
  | HTTPNodeConfig
  | LLMNodeConfig
  | TransformNodeConfig
  | FunctionCallNodeConfig;

// Node type enum
export enum NodeType {
  HTTP = "http",
  LLM = "llm",
  TRANSFORM = "transform",
  FUNCTION_CALL = "function_call",
}

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
};
