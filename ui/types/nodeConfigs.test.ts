import { describe, expect, it } from 'vitest';
import { NodeType, isValidNodeType } from '@/types';
import {
  BUILTIN_FUNCTIONS,
  DEFAULT_NODE_CONFIGS,
  DEFAULT_TOOL_CALL_CONFIG,
  FUNCTION_TYPES,
  HTTP_METHODS,
  LLM_PROVIDER_MODELS,
  NODE_OUTPUT_SCHEMAS,
  NODE_TYPE_METADATA,
  NodeTypeValues,
  TELEGRAM_FILE_SOURCES,
  TELEGRAM_MESSAGE_TYPES,
  TELEGRAM_PARSE_MODES,
  TOOL_CALL_MODES,
  TRANSFORM_TYPES,
} from './nodeConfigs';

/**
 * Node types without static defaults: comments are canvas notes, and OpenAPI
 * nodes get their config from the selected API operation (nodeConfigRegistry).
 */
const DYNAMIC_TYPES = new Set<string>([NodeType.COMMENT, NodeType.OPENAPI]);
const STATIC_TYPES = Object.values(NodeType).filter((type) => !DYNAMIC_TYPES.has(type));

describe('node configuration tables', () => {
  it('cover every executable node type', () => {
    STATIC_TYPES.forEach((type) => {
      expect(DEFAULT_NODE_CONFIGS, type).toHaveProperty(type);
      expect(NODE_TYPE_METADATA, type).toHaveProperty(type);
      expect(NODE_OUTPUT_SCHEMAS, type).toHaveProperty(type);
    });
  });

  it('are keyed by valid node types only', () => {
    [DEFAULT_NODE_CONFIGS, NODE_TYPE_METADATA, NODE_OUTPUT_SCHEMAS].forEach((table) => {
      Object.keys(table).forEach((key) => expect(isValidNodeType(key), key).toBe(true));
    });
  });

  it('have metadata that matches its key and is complete', () => {
    const categories = ['triggers', 'actions', 'logic', 'telegram', 'adapters', 'storage'];
    Object.entries(NODE_TYPE_METADATA).forEach(([key, meta]) => {
      expect(meta.type).toBe(key);
      expect(meta.label).not.toBe('');
      expect(meta.description).not.toBe('');
      expect(meta.icon).not.toBe('');
      expect(meta.color).toMatch(/^#[0-9A-Fa-f]{6}$/);
      expect(categories).toContain(meta.category);
    });
    const labels = Object.values(NODE_TYPE_METADATA).map((m) => m.label);
    expect(new Set(labels).size).toBe(labels.length);
  });

  it('use default values that are valid choices', () => {
    const http = DEFAULT_NODE_CONFIGS[NodeType.HTTP] as { method: string };
    expect(HTTP_METHODS).toContain(http.method);
    const transform = DEFAULT_NODE_CONFIGS[NodeType.TRANSFORM] as { type: string };
    expect(TRANSFORM_TYPES).toContain(transform.type);
    const telegram = DEFAULT_NODE_CONFIGS[NodeType.TELEGRAM] as { message_type: string; parse_mode: string };
    expect(TELEGRAM_MESSAGE_TYPES).toContain(telegram.message_type);
    expect(TELEGRAM_PARSE_MODES).toContain(telegram.parse_mode);
    // The model picker only offers the models of the provider, so the default must be one of them.
    const llm = DEFAULT_NODE_CONFIGS[NodeType.LLM] as { provider: string; model: string };
    expect(LLM_PROVIDER_MODELS[llm.provider]).toContain(llm.model);
  });

  it('keep NodeTypeValues as an alias of NodeType', () => {
    expect(NodeTypeValues).toBe(NodeType);
  });
});

describe('option lists', () => {
  it('have no duplicates and no empty entries', () => {
    const lists: readonly (readonly string[])[] = [
      HTTP_METHODS,
      TRANSFORM_TYPES,
      TELEGRAM_MESSAGE_TYPES,
      TELEGRAM_PARSE_MODES,
      TELEGRAM_FILE_SOURCES,
      TOOL_CALL_MODES,
      FUNCTION_TYPES,
      BUILTIN_FUNCTIONS,
      ...Object.values(LLM_PROVIDER_MODELS),
    ];
    lists.forEach((list) => {
      expect(list.length).toBeGreaterThan(0);
      expect(new Set(list).size).toBe(list.length);
      list.forEach((item) => expect(item.trim()).toBe(item));
    });
  });

  it('use a tool-call default within the declared modes', () => {
    expect(TOOL_CALL_MODES).toContain(DEFAULT_TOOL_CALL_CONFIG.mode);
    expect(DEFAULT_TOOL_CALL_CONFIG.max_iterations).toBeGreaterThan(0);
    expect(DEFAULT_TOOL_CALL_CONFIG.total_timeout).toBeGreaterThanOrEqual(DEFAULT_TOOL_CALL_CONFIG.timeout_per_tool!);
  });
});
