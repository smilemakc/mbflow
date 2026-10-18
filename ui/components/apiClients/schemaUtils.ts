import type {ApiOperation, ApiResponse, JsonSchema} from '@/services/apiClientService';

const REF_PREFIX = '#/components/schemas/';
const MAX_REF_HOPS = 32;
const MAX_EXAMPLE_DEPTH = 6;

type Schemas = Record<string, JsonSchema>;

/** Name of the component a "$ref" points to, with JSON Pointer escapes decoded. */
export const refName = (schema?: JsonSchema): string | undefined => {
    const ref = schema?.$ref;
    if (typeof ref !== 'string' || !ref.startsWith(REF_PREFIX)) return undefined;
    return ref.slice(REF_PREFIX.length).replace(/~1/g, '/').replace(/~0/g, '~');
};

/** Follows "$ref" into the client schemas; undefined for dangling references. */
export const resolveSchema = (schema: JsonSchema | undefined, schemas: Schemas): JsonSchema | undefined => {
    let current = schema;
    for (let i = 0; i < MAX_REF_HOPS && current; i++) {
        if (typeof current.$ref !== 'string') return current;
        const name = refName(current);
        if (!name) return undefined;
        current = schemas[name];
    }
    return undefined;
};

/** Resolves references and merges allOf parts, so object properties can be listed. */
export const flattenSchema = (schema: JsonSchema | undefined, schemas: Schemas, depth = 0): JsonSchema | undefined => {
    const resolved = resolveSchema(schema, schemas);
    if (!resolved || !Array.isArray(resolved.allOf) || depth > 16) return resolved;

    const {allOf, ...merged} = resolved;
    const properties: Record<string, JsonSchema> = {...(resolved.properties ?? {})};
    const required = new Set<string>(resolved.required ?? []);
    for (const part of allOf as JsonSchema[]) {
        const flat = flattenSchema(part, schemas, depth + 1);
        if (!flat) continue;
        Object.entries(flat.properties ?? {}).forEach(([key, value]) => {
            if (!(key in properties)) properties[key] = value as JsonSchema;
        });
        (flat.required ?? []).forEach((name: string) => required.add(name));
        if (!merged.type && flat.type) merged.type = flat.type;
        if (!merged.description && flat.description) merged.description = flat.description;
    }
    if (Object.keys(properties).length > 0) merged.properties = properties;
    if (required.size > 0) merged.required = [...required];
    return merged;
};

/** Declared type, skipping "null" in OpenAPI 3.1 type arrays, or a type inferred from keywords. */
export const schemaType = (schema?: JsonSchema): string => {
    if (!schema) return '';
    const type = schema.type;
    if (typeof type === 'string') return type;
    if (Array.isArray(type)) return type.find((t: string) => t !== 'null') ?? type[0] ?? '';
    if (schema.properties) return 'object';
    if (schema.items) return 'array';
    return '';
};

/** Short human-readable type, e.g. "Pet", "array<string>", "string (uuid)". */
export const describeSchema = (schema: JsonSchema | undefined, schemas: Schemas, depth = 0): string => {
    const name = refName(schema);
    if (name) return name;
    const resolved = flattenSchema(schema, schemas);
    if (!resolved) return 'any';
    const alternatives: JsonSchema[] | undefined = resolved.oneOf ?? resolved.anyOf;
    if (alternatives?.length && depth < 3) {
        return alternatives.slice(0, 3).map((alt) => describeSchema(alt, schemas, depth + 1)).join(' | ');
    }
    const type = schemaType(resolved);
    if (type === 'array') {
        return `array<${depth < 3 ? describeSchema(resolved.items, schemas, depth + 1) : '…'}>`;
    }
    if (type && resolved.format) return `${type} (${resolved.format})`;
    return type || 'any';
};

export interface SchemaField {
    name: string;
    schema: JsonSchema;
    required: boolean;
}

/**
 * Writable properties of an object schema, required ones first.
 * Returns null when the schema is not an object with known properties.
 */
export const objectFields = (schema: JsonSchema | undefined, schemas: Schemas): SchemaField[] | null => {
    const resolved = flattenSchema(schema, schemas);
    if (!resolved || schemaType(resolved) !== 'object' || !resolved.properties) return null;
    const required = new Set<string>(resolved.required ?? []);
    return Object.entries(resolved.properties as Record<string, JsonSchema>)
        .filter(([, prop]) => !flattenSchema(prop, schemas)?.readOnly)
        .map(([name, prop]) => ({name, schema: prop, required: required.has(name)}))
        .sort((a, b) => Number(b.required) - Number(a.required));
};

const formatExample = (format?: string): string => {
    switch (format) {
        case 'date-time':
            return '2024-01-01T00:00:00Z';
        case 'date':
            return '2024-01-01';
        case 'email':
            return 'user@example.com';
        case 'uuid':
            return '00000000-0000-0000-0000-000000000000';
        case 'uri':
        case 'url':
            return 'https://example.com';
        default:
            return '';
    }
};

/** Example value built from examples, defaults and types; recursive references stop. */
export const buildExample = (
    schema: JsonSchema | undefined,
    schemas: Schemas,
    depth = 0,
    seen: ReadonlySet<string> = new Set(),
): unknown => {
    if (depth > MAX_EXAMPLE_DEPTH) return undefined;
    const name = refName(schema);
    if (name) {
        if (seen.has(name)) return undefined;
        seen = new Set([...seen, name]);
    }
    const resolved = flattenSchema(schema, schemas);
    if (!resolved) return null;
    if (resolved.example !== undefined) return resolved.example;
    if (resolved.default !== undefined) return resolved.default;
    if (resolved.const !== undefined) return resolved.const;
    if (Array.isArray(resolved.enum) && resolved.enum.length > 0) return resolved.enum[0];
    const alternative = resolved.oneOf?.[0] ?? resolved.anyOf?.[0];
    if (alternative) return buildExample(alternative, schemas, depth + 1, seen);

    switch (schemaType(resolved)) {
        case 'object': {
            const result: Record<string, unknown> = {};
            Object.entries((resolved.properties ?? {}) as Record<string, JsonSchema>).forEach(([key, prop]) => {
                if (flattenSchema(prop, schemas)?.readOnly) return;
                const value = buildExample(prop, schemas, depth + 1, seen);
                if (value !== undefined) result[key] = value;
            });
            return result;
        }
        case 'array': {
            const item = buildExample(resolved.items, schemas, depth + 1, seen);
            return item === undefined ? [] : [item];
        }
        case 'integer':
        case 'number':
            return 0;
        case 'boolean':
            return false;
        case 'string':
            return formatExample(resolved.format);
        default:
            return null;
    }
};

/** The first 2xx response, or "default". */
export const successResponse = (operation: ApiOperation): ApiResponse | undefined =>
    operation.responses?.find((r) => r.status.startsWith('2')) ??
    operation.responses?.find((r) => r.status === 'default');

export const operationLabel = (operation: ApiOperation): string =>
    operation.summary || `${operation.method} ${operation.path}`;

export const operationMatches = (operation: ApiOperation, query: string): boolean => {
    const q = query.trim().toLowerCase();
    if (!q) return true;
    return [operation.id, operation.path, operation.summary, operation.method, ...(operation.tags ?? [])]
        .some((value) => value?.toLowerCase().includes(q));
};

/** Groups operations by their first tag, keeping the order in which tags appear. */
export const groupByTag = (operations: ApiOperation[], untagged: string): [string, ApiOperation[]][] => {
    const groups = new Map<string, ApiOperation[]>();
    operations.forEach((operation) => {
        const tag = operation.tags?.[0] || untagged;
        const list = groups.get(tag) ?? [];
        list.push(operation);
        groups.set(tag, list);
    });
    return [...groups.entries()];
};

/** Replaces "{name}" placeholders in a translated string. */
export const format = (template: string, values: Record<string, string | number>): string =>
    template.replace(/\{(\w+)\}/g, (match, key) => (key in values ? String(values[key]) : match));
