import type {
  ParsedVariable,
  TemplateOptions,
  VariableContext,
  VariableType,
} from "@/types/template";

// Template pattern: {{type.path.to.value}}
const TEMPLATE_PATTERN = /\{\{([^}]+)\}\}/g;

/**
 * Parse template string and extract all variables
 */
export function parseTemplate(template: string): ParsedVariable[] {
  const variables: ParsedVariable[] = [];
  const matches = template.matchAll(TEMPLATE_PATTERN);

  for (const match of matches) {
    const fullMatch = match[0];
    const content = match[1]?.trim();
    const parts = content?.split(".");
    if (!parts) continue;
    if (parts.length < 2) {
      continue; // Invalid format
    }

    const type = parts[0] as VariableType;
    const path = parts.slice(1).join(".");

    if (type === "env" || type === "input") {
      variables.push({ fullMatch, type, path });
    }
  }

  return variables;
}

/**
 * Check if string contains templates
 */
export function hasTemplates(str: string): boolean {
  return TEMPLATE_PATTERN.test(str);
}

/**
 * Validate template syntax
 */
export function validateTemplateSyntax(template: string): {
  valid: boolean;
  error?: string;
} {
  const matches = template.matchAll(TEMPLATE_PATTERN);

  for (const match of matches) {
    const content = match[1]?.trim();
    const parts = content?.split(".");
    if (!parts) continue;
    if (parts.length < 2) {
      return {
        valid: false,
        error: `Invalid template "${match[0]}": must be {{type.path}}`,
      };
    }

    const type = parts[0];
    if (type !== "env" && type !== "input") {
      return {
        valid: false,
        error: `Invalid variable type "${type}" in "${match[0]}": must be "env" or "input"`,
      };
    }
  }

  return { valid: true };
}

/**
 * Resolve path in object (supports dot notation and array indexing)
 * Examples:
 *   - "user.name" -> obj.user.name
 *   - "items[0]" -> obj.items[0]
 *   - "users[0].email" -> obj.users[0].email
 */
export function resolvePath(obj: any, path: string): any {
  if (!path) return obj;

  const parts = path.split(".");
  let current = obj;

  for (const part of parts) {
    if (current === undefined || current === null) {
      return undefined;
    }

    // Handle array access: items[0]
    const arrayMatch = part.match(/^(\w+)\[(\d+)\]$/);
    if (arrayMatch) {
      const [, key, index] = arrayMatch;
      if (!key) continue;
      current = current[key];
      if (Array.isArray(current)) {
        if (!index) continue;
        current = current[parseInt(index)];
      } else {
        return undefined;
      }
    } else {
      current = current[part];
    }
  }

  return current;
}

/**
 * Convert value to string for template substitution
 */
function valueToString(value: any): string {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

/**
 * Resolve a single variable reference
 */
function resolveVariable(
  varRef: ParsedVariable,
  context: VariableContext,
  options: TemplateOptions,
): string {
  const { type, path } = varRef;

  let value: any;

  if (type === "env") {
    // ExecutionVars take precedence over WorkflowVars
    value =
      resolvePath(context.executionVars, path) ??
      resolvePath(context.workflowVars, path);
  } else if (type === "input") {
    value = resolvePath(context.inputVars, path);
  }

  if (value === undefined) {
    if (options.strictMode) {
      throw new Error(`Variable not found: ${varRef.fullMatch}`);
    }
    if (options.placeholderOnMissing) {
      return varRef.fullMatch; // Keep {{...}} placeholder
    }
    return "";
  }

  return valueToString(value);
}

/**
 * Resolve template string with variable context
 */
export function resolveTemplate(
  template: string,
  context: VariableContext,
  options: TemplateOptions = {},
): string {
  const variables = parseTemplate(template);

  let result = template;

  for (const varRef of variables) {
    const value = resolveVariable(varRef, context, options);
    result = result.replace(varRef.fullMatch, value);
  }

  return result;
}

/**
 * Resolve all templates in an object recursively
 */
export function resolveTemplateObject(
  obj: any,
  context: VariableContext,
  options: TemplateOptions = {},
): any {
  if (typeof obj === "string") {
    return resolveTemplate(obj, context, options);
  }

  if (Array.isArray(obj)) {
    return obj.map((item) => resolveTemplateObject(item, context, options));
  }

  if (obj !== null && typeof obj === "object") {
    const result: Record<string, any> = {};
    for (const [key, value] of Object.entries(obj)) {
      result[key] = resolveTemplateObject(value, context, options);
    }
    return result;
  }

  return obj;
}

/**
 * Extract all unique variable paths from template
 * Useful for autocomplete
 */
export function extractVariablePaths(template: string): {
  env: string[];
  input: string[];
} {
  const variables = parseTemplate(template);
  const env = new Set<string>();
  const input = new Set<string>();

  for (const varRef of variables) {
    if (varRef.type === "env") {
      env.add(varRef.path);
    } else if (varRef.type === "input") {
      input.add(varRef.path);
    }
  }

  return {
    env: Array.from(env),
    input: Array.from(input),
  };
}
