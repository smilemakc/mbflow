/**
 * Node configuration types for React application
 * Ported from Vue: /mbflow-ui/src/types/nodes.ts
 * Backend reference: internal/domain/workflow/node.go
 */

import { NodeType } from '@/types';

export interface BaseNodeConfig {
  [key: string]: any;
}

// HTTP Node
export interface HTTPNodeConfig extends BaseNodeConfig {
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS";
  headers?: Record<string, string>;
  body?: string;
  timeout_seconds?: number;
  retry_count?: number;
  follow_redirects?: boolean;
}

// Tool Calling types
export type ToolCallMode = "auto" | "manual";
export type FunctionType = "builtin" | "sub_workflow" | "custom_code" | "openapi";

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
  parameters?: Record<string, any>;

  // Builtin function
  builtin_name?: string;

  // Sub-workflow
  workflow_id?: string;
  input_mapping?: Record<string, string>;
  output_extractor?: string;

  // Custom code
  language?: "javascript" | "python";
  code?: string;

  // OpenAPI
  openapi_spec?: string;
  operation_id?: string;
  base_url?: string;
  auth_config?: Record<string, any>;
}

// LLM Node
export interface LLMNodeConfig extends BaseNodeConfig {
  provider: "openai" | "anthropic" | "google" | "azure" | "ollama";
  model: string;
  api_key: string;

  instruction?: string;
  prompt: string;

  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  frequency_penalty?: number;
  presence_penalty?: number;
  stop_sequences?: string[];
  response_format?: "text" | "json";

  timeout_seconds?: number;
  retry_count?: number;

  stream?: boolean;

  // Tool calling
  tool_call_config?: ToolCallConfig;
  functions?: FunctionDefinition[];
}

// Transform Node
export interface TransformNodeConfig extends BaseNodeConfig {
  language: "jq" | "javascript";
  expression: string;
  timeout_seconds?: number;
}

// Function Call Node
export interface FunctionCallNodeConfig extends BaseNodeConfig {
  function_name: string;
  arguments?: Record<string, any>;
  timeout_seconds?: number;
}

// Telegram Nodes
export interface TelegramNodeConfig extends BaseNodeConfig {
  bot_token: string;
  chat_id: string;
  message_type: "text" | "photo" | "document" | "audio" | "video";
  text?: string;
  parse_mode?: "Markdown" | "MarkdownV2" | "HTML";
  disable_web_page_preview?: boolean;
  disable_notification?: boolean;
  protect_content?: boolean;

  file_source?: "base64" | "url" | "file_id";
  file_data?: string;
  file_name?: string;

  timeout_seconds?: number;
}

export interface TelegramDownloadNodeConfig extends BaseNodeConfig {
  bot_token: string;
  file_id: string;
  output_format?: "base64" | "url";
  timeout?: number;
}

export interface TelegramParseNodeConfig extends BaseNodeConfig {
  extract_files?: boolean;
  extract_commands?: boolean;
  extract_entities?: boolean;
}

export interface TelegramCallbackNodeConfig extends BaseNodeConfig {
  bot_token: string;
  callback_query_id: string;
  text?: string;
  show_alert?: boolean;
  url?: string;
  cache_time?: number;
  timeout?: number;
}

// File Storage Node
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

// Conditional Node
export interface ConditionalNodeConfig extends BaseNodeConfig {
  condition: string;
  true_branch?: string;
  false_branch?: string;
}

// Merge Node
export interface MergeNodeConfig extends BaseNodeConfig {
  merge_strategy?: "first" | "last" | "all" | "custom";
  custom_expression?: string;
}

// Delay Node
export interface DelayNodeConfig extends BaseNodeConfig {
  duration: number;
  unit: "seconds" | "minutes" | "hours";
  description?: string;
}

// Adapter Nodes
export interface Base64ToBytesNodeConfig extends BaseNodeConfig {
  encoding: "standard" | "url" | "raw_standard" | "raw_url";
  output_format: "raw" | "hex";
}

export interface BytesToBase64NodeConfig extends BaseNodeConfig {
  encoding: "standard" | "url" | "raw_standard" | "raw_url";
  line_length?: number;
}

export interface StringToJsonNodeConfig extends BaseNodeConfig {
  strict_mode?: boolean;
  trim_whitespace?: boolean;
}

export interface JsonToStringNodeConfig extends BaseNodeConfig {
  pretty?: boolean;
  indent?: string;
  escape_html?: boolean;
  sort_keys?: boolean;
}

export interface BytesToJsonNodeConfig extends BaseNodeConfig {
  encoding: "utf-8" | "utf-16" | "latin1";
  validate_json?: boolean;
}

export interface FileToBytesNodeConfig extends BaseNodeConfig {
  storage_id?: string;
  file_id: string;
  output_format: "raw" | "base64";
}

export interface BytesToFileNodeConfig extends BaseNodeConfig {
  storage_id?: string;
  file_name: string;
  mime_type?: string;
  access_scope?: "workflow" | "edge" | "result";
  ttl?: number;
  tags?: string[];
}

// HTML Clean Node
export interface HTMLCleanNodeConfig extends BaseNodeConfig {
  input_key?: string;  // Key to extract content from input (optional, auto-detects common keys)
  output_format: "text" | "html" | "both";
  extract_metadata: boolean;
  preserve_links: boolean;
  max_length?: number;
}

// RSS Parser Node
export interface RSSParserNodeConfig extends BaseNodeConfig {
  url: string;
  maxItems?: number;
  includeContent?: boolean;
}

// CSV to JSON Node
export interface CSVToJSONNodeConfig extends BaseNodeConfig {
  delimiter: string;
  has_header: boolean;
  custom_headers?: string[];
  trim_spaces: boolean;
  skip_empty_rows: boolean;
  input_key?: string;
}

// Union type of all node configs
export type NodeConfig =
  | HTTPNodeConfig
  | LLMNodeConfig
  | TransformNodeConfig
  | FunctionCallNodeConfig
  | TelegramNodeConfig
  | TelegramDownloadNodeConfig
  | TelegramParseNodeConfig
  | TelegramCallbackNodeConfig
  | FileStorageNodeConfig
  | ConditionalNodeConfig
  | MergeNodeConfig
  | DelayNodeConfig
  | Base64ToBytesNodeConfig
  | BytesToBase64NodeConfig
  | StringToJsonNodeConfig
  | JsonToStringNodeConfig
  | BytesToJsonNodeConfig
  | FileToBytesNodeConfig
  | BytesToFileNodeConfig
  | HTMLCleanNodeConfig
  | RSSParserNodeConfig
  | CSVToJSONNodeConfig;

// NodeTypeValues is deprecated - use NodeType enum from '@/types' instead
// Kept as alias for backward compatibility during migration
export const NodeTypeValues = NodeType;

// Default configurations - keyed by NodeType enum values (which are backend type strings)
export const DEFAULT_NODE_CONFIGS: Record<string, NodeConfig> = {
  [NodeType.HTTP]: {  // 'http'
    url: "",
    method: "GET",
    headers: {},
    timeout_seconds: 30,
    retry_count: 0,
    follow_redirects: true,
  },
  [NodeType.LLM]: {  // 'llm'
    provider: "openai",
    model: "gpt-4",
    api_key: "",
    prompt: "",
    temperature: 0.7,
    max_tokens: 1000,
  },
  [NodeType.TRANSFORM]: {  // 'transform'
    language: "jq",
    expression: ".",
    timeout_seconds: 10,
  },
  [NodeType.FUNCTION_CALL]: {  // 'function_call'
    function_name: "",
    arguments: {},
    timeout_seconds: 30,
  },
  [NodeType.TELEGRAM]: {  // 'telegram'
    bot_token: "",
    chat_id: "",
    message_type: "text",
    text: "",
    parse_mode: "HTML",
    timeout_seconds: 30,
  },
  [NodeType.TELEGRAM_DOWNLOAD]: {  // 'telegram_download'
    bot_token: "",
    file_id: "",
    output_format: "base64",
    timeout: 60,
  },
  [NodeType.TELEGRAM_PARSE]: {  // 'telegram_parse'
    extract_files: true,
    extract_commands: true,
    extract_entities: false,
  },
  [NodeType.TELEGRAM_CALLBACK]: {  // 'telegram_callback'
    bot_token: "",
    callback_query_id: "",
    text: "",
    show_alert: false,
    cache_time: 0,
  },
  [NodeType.FILE_STORAGE]: {  // 'file_storage'
    action: "store",
    storage_id: "",
    access_scope: "workflow",
    ttl: 0,
  },
  [NodeType.CONDITIONAL]: {  // 'conditional'
    condition: "{{input.value}} > 0",
  },
  [NodeType.MERGE]: {  // 'merge'
    merge_strategy: "all",
  },
  [NodeType.DELAY]: {  // 'delay'
    duration: 1,
    unit: "seconds",
    description: "",
  },
  [NodeType.BASE64_TO_BYTES]: {  // 'base64_to_bytes'
    encoding: "standard",
    output_format: "raw",
  },
  [NodeType.BYTES_TO_BASE64]: {  // 'bytes_to_base64'
    encoding: "standard",
    line_length: 0,
  },
  [NodeType.STRING_TO_JSON]: {  // 'string_to_json'
    strict_mode: true,
    trim_whitespace: true,
  },
  [NodeType.JSON_TO_STRING]: {  // 'json_to_string'
    pretty: false,
    indent: "  ",
    escape_html: true,
    sort_keys: false,
  },
  [NodeType.BYTES_TO_JSON]: {  // 'bytes_to_json'
    encoding: "utf-8",
    validate_json: true,
  },
  [NodeType.FILE_TO_BYTES]: {  // 'file_to_bytes'
    storage_id: "default",
    file_id: "",
    output_format: "base64",
  },
  [NodeType.BYTES_TO_FILE]: {  // 'bytes_to_file'
    storage_id: "default",
    file_name: "",
    access_scope: "workflow",
    ttl: 0,
    tags: [],
  },
  [NodeType.HTML_CLEAN]: {  // 'html_clean'
    input_key: "",
    output_format: "both",
    extract_metadata: true,
    preserve_links: false,
    max_length: 0,
  },
  [NodeType.RSS_PARSER]: {  // 'rss_parser'
    url: "",
    maxItems: 0,
    includeContent: false,
  },
  [NodeType.CSV_TO_JSON]: {  // 'csv_to_json'
    delimiter: ",",
    has_header: true,
    custom_headers: [],
    trim_spaces: true,
    skip_empty_rows: true,
    input_key: "",
  },
};

// LLM Provider models
export const LLM_PROVIDER_MODELS: Record<string, string[]> = {
  openai: [
    "gpt-4",
    "gpt-4-turbo",
    "gpt-4-turbo-preview",
    "gpt-4o",
    "gpt-4o-mini",
    "gpt-3.5-turbo",
    "gpt-3.5-turbo-16k",
  ],
  anthropic: [
    "claude-3-opus-20240229",
    "claude-3-sonnet-20240229",
    "claude-3-haiku-20240307",
    "claude-3-5-sonnet-20241022",
    "claude-2.1",
  ],
  google: ["gemini-pro", "gemini-pro-vision", "gemini-1.5-pro"],
  azure: ["gpt-4", "gpt-35-turbo"],
  ollama: ["llama2", "llama3", "mistral", "codellama", "mixtral"],
};

// Constants
export const HTTP_METHODS = [
  "GET",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "HEAD",
  "OPTIONS",
] as const;

export const TRANSFORM_LANGUAGES = ["jq", "javascript"] as const;

export const TELEGRAM_MESSAGE_TYPES = [
  "text",
  "photo",
  "document",
  "audio",
  "video",
] as const;

export const TELEGRAM_PARSE_MODES = ["Markdown", "MarkdownV2", "HTML"] as const;
export const TELEGRAM_FILE_SOURCES = ["base64", "url", "file_id"] as const;

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
  type: string;
  label: string;
  description: string;
  icon: string;
  color: string;
  category: "triggers" | "actions" | "logic" | "telegram" | "adapters" | "storage";
}

export const NODE_TYPE_METADATA: Record<string, NodeTypeMetadata> = {
  [NodeType.HTTP]: {  // 'http'
    type: NodeType.HTTP,
    label: "HTTP Request",
    description: "Make HTTP/HTTPS requests to external APIs",
    icon: "Globe",
    color: "#10B981",
    category: "actions",
  },
  [NodeType.LLM]: {  // 'llm'
    type: NodeType.LLM,
    label: "LLM Call",
    description: "Call language models (OpenAI, Anthropic, etc.)",
    icon: "Sparkles",
    color: "#8B5CF6",
    category: "actions",
  },
  [NodeType.TRANSFORM]: {  // 'transform'
    type: NodeType.TRANSFORM,
    label: "Transform",
    description: "Transform data using jq or JavaScript",
    icon: "Zap",
    color: "#F59E0B",
    category: "logic",
  },
  [NodeType.FUNCTION_CALL]: {  // 'function_call'
    type: NodeType.FUNCTION_CALL,
    label: "Function Call",
    description: "Call custom functions or tools",
    icon: "Code",
    color: "#3B82F6",
    category: "actions",
  },
  [NodeType.TELEGRAM]: {  // 'telegram'
    type: NodeType.TELEGRAM,
    label: "Telegram",
    description: "Send messages via Telegram Bot API",
    icon: "Send",
    color: "#0EA5E9",
    category: "telegram",
  },
  [NodeType.TELEGRAM_DOWNLOAD]: {  // 'telegram_download'
    type: NodeType.TELEGRAM_DOWNLOAD,
    label: "TG Download",
    description: "Download files from Telegram by file_id",
    icon: "Download",
    color: "#0EA5E9",
    category: "telegram",
  },
  [NodeType.TELEGRAM_PARSE]: {  // 'telegram_parse'
    type: NodeType.TELEGRAM_PARSE,
    label: "TG Parse",
    description: "Parse Telegram updates and extract data",
    icon: "FileSearch",
    color: "#0EA5E9",
    category: "telegram",
  },
  [NodeType.TELEGRAM_CALLBACK]: {  // 'telegram_callback'
    type: NodeType.TELEGRAM_CALLBACK,
    label: "TG Callback",
    description: "Answer Telegram callback queries",
    icon: "CheckCircle",
    color: "#0EA5E9",
    category: "telegram",
  },
  [NodeType.FILE_STORAGE]: {  // 'file_storage'
    type: NodeType.FILE_STORAGE,
    label: "File Storage",
    description: "Store, retrieve, and manage files",
    icon: "Folder",
    color: "#14B8A6",
    category: "storage",
  },
  [NodeType.CONDITIONAL]: {  // 'conditional'
    type: NodeType.CONDITIONAL,
    label: "Conditional",
    description: "Branch workflow based on conditions",
    icon: "GitBranch",
    color: "#EC4899",
    category: "logic",
  },
  [NodeType.MERGE]: {  // 'merge'
    type: NodeType.MERGE,
    label: "Merge",
    description: "Merge results from multiple nodes",
    icon: "GitMerge",
    color: "#A855F7",
    category: "logic",
  },
  [NodeType.DELAY]: {  // 'delay'
    type: NodeType.DELAY,
    label: "Delay",
    description: "Add a time delay before proceeding",
    icon: "Clock",
    color: "#06B6D4",
    category: "logic",
  },
  [NodeType.BASE64_TO_BYTES]: {  // 'base64_to_bytes'
    type: NodeType.BASE64_TO_BYTES,
    label: "Base64 → Bytes",
    description: "Decode base64 string to bytes",
    icon: "Unlock",
    color: "#EF4444",
    category: "adapters",
  },
  [NodeType.BYTES_TO_BASE64]: {  // 'bytes_to_base64'
    type: NodeType.BYTES_TO_BASE64,
    label: "Bytes → Base64",
    description: "Encode bytes to base64 string",
    icon: "Lock",
    color: "#F59E0B",
    category: "adapters",
  },
  [NodeType.STRING_TO_JSON]: {  // 'string_to_json'
    type: NodeType.STRING_TO_JSON,
    label: "String → JSON",
    description: "Parse JSON string to object",
    icon: "Braces",
    color: "#8B5CF6",
    category: "adapters",
  },
  [NodeType.JSON_TO_STRING]: {  // 'json_to_string'
    type: NodeType.JSON_TO_STRING,
    label: "JSON → String",
    description: "Serialize JSON to string",
    icon: "FileText",
    color: "#EC4899",
    category: "adapters",
  },
  [NodeType.BYTES_TO_JSON]: {  // 'bytes_to_json'
    type: NodeType.BYTES_TO_JSON,
    label: "Bytes → JSON",
    description: "Decode bytes to JSON with encoding detection",
    icon: "Box",
    color: "#06B6D4",
    category: "adapters",
  },
  [NodeType.FILE_TO_BYTES]: {  // 'file_to_bytes'
    type: NodeType.FILE_TO_BYTES,
    label: "File → Bytes",
    description: "Read file from storage as bytes",
    icon: "FileDown",
    color: "#10B981",
    category: "adapters",
  },
  [NodeType.BYTES_TO_FILE]: {  // 'bytes_to_file'
    type: NodeType.BYTES_TO_FILE,
    label: "Bytes → File",
    description: "Save bytes to file storage",
    icon: "FileUp",
    color: "#14B8A6",
    category: "adapters",
  },
  [NodeType.HTML_CLEAN]: {  // 'html_clean'
    type: NodeType.HTML_CLEAN,
    label: "HTML Clean",
    description: "Extract readable content from HTML, removing scripts, styles, and boilerplate",
    icon: "FileText",
    color: "#F97316",
    category: "actions",
  },
  [NodeType.RSS_PARSER]: {  // 'rss_parser'
    type: NodeType.RSS_PARSER,
    label: "RSS Parser",
    description: "Parse RSS/Atom feeds and extract structured data",
    icon: "Rss",
    color: "#F97316",
    category: "actions",
  },
  [NodeType.CSV_TO_JSON]: {  // 'csv_to_json'
    type: NodeType.CSV_TO_JSON,
    label: "CSV → JSON",
    description: "Convert CSV data to JSON array of objects",
    icon: "Table",
    color: "#06B6D4",
    category: "adapters",
  },
};

// Node output schemas for variable autocomplete
export const NODE_OUTPUT_SCHEMAS: Record<string, Record<string, any>> = {
  [NodeType.HTTP]: {  // 'http'
    status: { type: "number", description: "HTTP status code" },
    headers: { type: "object", description: "Response headers" },
    body: { type: "any", description: "Response body (parsed if JSON)" },
    duration_ms: { type: "number", description: "Request duration in milliseconds" },
  },
  [NodeType.LLM]: {  // 'llm'
    content: { type: "string", description: "Generated text response" },
    model: { type: "string", description: "Model used" },
    usage: {
      type: "object",
      description: "Token usage statistics",
      properties: {
        prompt_tokens: { type: "number" },
        completion_tokens: { type: "number" },
        total_tokens: { type: "number" },
      },
    },
    tool_calls: { type: "array", description: "Tool calls made by the model" },
  },
  [NodeType.TRANSFORM]: {  // 'transform'
    result: { type: "any", description: "Transformed data" },
  },
  [NodeType.FUNCTION_CALL]: {  // 'function_call'
    result: { type: "any", description: "Function execution result" },
  },
  [NodeType.TELEGRAM]: {  // 'telegram'
    message_id: { type: "number", description: "Sent message ID" },
    chat: { type: "object", description: "Chat information" },
    date: { type: "number", description: "Message timestamp" },
  },
  [NodeType.TELEGRAM_DOWNLOAD]: {  // 'telegram_download'
    file_path: { type: "string", description: "Downloaded file path" },
    file_size: { type: "number", description: "File size in bytes" },
    file_data: { type: "string", description: "File data (base64 or URL)" },
  },
  [NodeType.TELEGRAM_PARSE]: {  // 'telegram_parse'
    message_type: { type: "string", description: "Type of message" },
    text: { type: "string", description: "Message text" },
    files: { type: "array", description: "Extracted files" },
    commands: { type: "array", description: "Extracted commands" },
    entities: { type: "array", description: "Message entities" },
  },
  [NodeType.TELEGRAM_CALLBACK]: {  // 'telegram_callback'
    success: { type: "boolean", description: "Whether callback was answered" },
  },
  [NodeType.FILE_STORAGE]: {  // 'file_storage'
    file_id: { type: "string", description: "File identifier" },
    file_name: { type: "string", description: "File name" },
    mime_type: { type: "string", description: "MIME type" },
    size: { type: "number", description: "File size in bytes" },
  },
  [NodeType.CONDITIONAL]: {  // 'conditional'
    result: { type: "boolean", description: "Condition evaluation result" },
    branch: { type: "string", description: "Selected branch (true/false)" },
  },
  [NodeType.MERGE]: {  // 'merge'
    merged: { type: "any", description: "Merged result from all inputs" },
  },
  [NodeType.DELAY]: {  // 'delay'
    delayed: { type: "boolean", description: "Whether delay was completed" },
    duration_ms: { type: "number", description: "Actual delay duration in milliseconds" },
  },
  [NodeType.BASE64_TO_BYTES]: {  // 'base64_to_bytes'
    bytes: { type: "string", description: "Decoded bytes" },
    size: { type: "number", description: "Byte array size" },
  },
  [NodeType.BYTES_TO_BASE64]: {  // 'bytes_to_base64'
    base64: { type: "string", description: "Encoded base64 string" },
  },
  [NodeType.STRING_TO_JSON]: {  // 'string_to_json'
    json: { type: "object", description: "Parsed JSON object" },
  },
  [NodeType.JSON_TO_STRING]: {  // 'json_to_string'
    string: { type: "string", description: "Serialized JSON string" },
  },
  [NodeType.BYTES_TO_JSON]: {  // 'bytes_to_json'
    json: { type: "object", description: "Decoded JSON object" },
    encoding: { type: "string", description: "Detected encoding" },
  },
  [NodeType.FILE_TO_BYTES]: {  // 'file_to_bytes'
    bytes: { type: "string", description: "File content as bytes" },
    size: { type: "number", description: "File size" },
  },
  [NodeType.BYTES_TO_FILE]: {  // 'bytes_to_file'
    file_id: { type: "string", description: "Stored file identifier" },
    file_name: { type: "string", description: "File name" },
    size: { type: "number", description: "File size" },
  },
  [NodeType.HTML_CLEAN]: {  // 'html_clean'
    text_content: { type: "string", description: "Cleaned plain text content" },
    html_content: { type: "string", description: "Cleaned minimal HTML content" },
    title: { type: "string", description: "Page title" },
    author: { type: "string", description: "Content author (if found)" },
    excerpt: { type: "string", description: "Content excerpt/summary" },
    site_name: { type: "string", description: "Website name" },
    length: { type: "number", description: "Text content length" },
    word_count: { type: "number", description: "Word count" },
    is_html: { type: "boolean", description: "Was input detected as HTML" },
    passthrough: { type: "boolean", description: "Was input returned as-is (not HTML)" },
  },
  [NodeType.RSS_PARSER]: {  // 'rss_parser'
    title: { type: "string", description: "Feed title" },
    description: { type: "string", description: "Feed description" },
    link: { type: "string", description: "Feed website link" },
    items: {
      type: "array",
      description: "Array of feed items",
      items: {
        title: { type: "string", description: "Item title" },
        link: { type: "string", description: "Item URL" },
        description: { type: "string", description: "Item summary" },
        content: { type: "string", description: "Full content (if includeContent=true)" },
        pubDate: { type: "string", description: "Publication date" },
        author: { type: "string", description: "Item author" },
        categories: { type: "array", description: "Item categories" },
        guid: { type: "string", description: "Item unique identifier" },
      }
    },
    item_count: { type: "number", description: "Number of items returned" },
    feed_type: { type: "string", description: "Feed format (rss or atom)" },
  },
  [NodeType.CSV_TO_JSON]: {  // 'csv_to_json'
    success: { type: "boolean", description: "Whether conversion was successful" },
    result: {
      type: "array",
      description: "Array of JSON objects",
      items: { type: "object", description: "Row as JSON object" }
    },
    row_count: { type: "number", description: "Number of data rows" },
    column_count: { type: "number", description: "Number of columns" },
    headers: { type: "array", description: "Column headers" },
    duration_ms: { type: "number", description: "Processing time in milliseconds" },
  },
};
