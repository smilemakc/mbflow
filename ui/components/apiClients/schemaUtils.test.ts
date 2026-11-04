import { describe, expect, it } from 'vitest';
import {
  buildExample,
  describeSchema,
  flattenSchema,
  format,
  groupByTag,
  objectFields,
  operationLabel,
  operationMatches,
  refName,
  resolveSchema,
  schemaType,
  successResponse,
} from './schemaUtils';
import { operation } from '@/test/adminFixtures';
import type { JsonSchema } from '@/services/apiClientService';

const ref = (name: string) => ({ $ref: `#/components/schemas/${name}` });

const SCHEMAS: Record<string, JsonSchema> = {
  Pet: {
    type: 'object',
    required: ['name'],
    properties: {
      id: { type: 'integer', readOnly: true },
      name: { type: 'string', example: 'Rex' },
      tag: { type: 'string' },
      owner: ref('Owner'),
      friends: { type: 'array', items: ref('Pet') },
    },
  },
  Owner: { type: 'object', properties: { email: { type: 'string', format: 'email' } } },
  Alias: ref('Pet'),
  'a/b~c': { type: 'string' },
  Loop: ref('Loop'),
  Base: { type: 'object', description: 'base', required: ['id'], properties: { id: { type: 'string' } } },
  Cat: {
    allOf: [ref('Base'), { properties: { id: { type: 'number' }, meow: { type: 'boolean' } }, required: ['meow'] }, ref('Missing')],
  },
};

describe('refName / resolveSchema', () => {
  it('decodes JSON pointer escapes and ignores foreign refs', () => {
    expect(refName({ $ref: '#/components/schemas/a~1b~0c' })).toBe('a/b~c');
    expect(refName({ $ref: 'other.yaml#/Pet' })).toBeUndefined();
    expect(refName({ $ref: 42 })).toBeUndefined();
    expect(refName(undefined)).toBeUndefined();
  });

  it('follows chains of references', () => {
    expect(resolveSchema(ref('Alias'), SCHEMAS)).toBe(SCHEMAS.Pet);
    expect(resolveSchema({ type: 'string' }, SCHEMAS)).toEqual({ type: 'string' });
    expect(resolveSchema(ref('Missing'), SCHEMAS)).toBeUndefined();
    expect(resolveSchema({ $ref: 'http://x' }, SCHEMAS)).toBeUndefined();
    expect(resolveSchema(ref('Loop'), SCHEMAS)).toBeUndefined();
    expect(resolveSchema(undefined, SCHEMAS)).toBeUndefined();
  });
});

describe('flattenSchema', () => {
  it('merges allOf parts, keeping the first definition of a property', () => {
    const cat = flattenSchema(ref('Cat'), SCHEMAS)!;
    expect(cat.allOf).toBeUndefined();
    expect(cat.type).toBe('object');
    expect(cat.description).toBe('base');
    expect(cat.properties).toEqual({ id: { type: 'string' }, meow: { type: 'boolean' } });
    expect(cat.required).toEqual(['id', 'meow']);
  });

  it('leaves an allOf without properties or requirements as is', () => {
    expect(flattenSchema({ type: 'string', description: 'x', allOf: [{ type: 'number', description: 'y' }] }, {})).toEqual({
      type: 'string',
      description: 'x',
    });
  });

  it('returns plain schemas unchanged', () => {
    expect(flattenSchema(ref('Owner'), SCHEMAS)).toBe(SCHEMAS.Owner);
  });
});

describe('schemaType', () => {
  it('reads declared and inferred types', () => {
    expect(schemaType(undefined)).toBe('');
    expect(schemaType({ type: 'string' })).toBe('string');
    expect(schemaType({ type: ['null', 'integer'] })).toBe('integer');
    expect(schemaType({ type: ['null'] })).toBe('null');
    expect(schemaType({ type: [] })).toBe('');
    expect(schemaType({ properties: {} })).toBe('object');
    expect(schemaType({ items: {} })).toBe('array');
    expect(schemaType({})).toBe('');
  });
});

describe('describeSchema', () => {
  it('describes references, arrays, formats and alternatives', () => {
    expect(describeSchema(ref('Pet'), SCHEMAS)).toBe('Pet');
    expect(describeSchema({ type: 'array', items: ref('Pet') }, SCHEMAS)).toBe('array<Pet>');
    expect(describeSchema({ type: 'string', format: 'uuid' }, SCHEMAS)).toBe('string (uuid)');
    expect(describeSchema({ oneOf: [{ type: 'string' }, ref('Owner')] }, SCHEMAS)).toBe('string | Owner');
    expect(describeSchema({ anyOf: [{ type: 'integer' }] }, SCHEMAS)).toBe('integer');
    expect(describeSchema({}, SCHEMAS)).toBe('any');
    expect(describeSchema({ $ref: 'http://x' }, SCHEMAS)).toBe('any');
    expect(describeSchema({ items: { items: { items: { items: { type: 'string' } } } } }, SCHEMAS)).toBe(
      'array<array<array<array<…>>>>',
    );
  });
});

describe('objectFields', () => {
  it('lists writable properties, required first', () => {
    const fields = objectFields(ref('Pet'), SCHEMAS)!;
    expect(fields.map((f) => [f.name, f.required])).toEqual([
      ['name', true],
      ['tag', false],
      ['owner', false],
      ['friends', false],
    ]);
  });

  it('returns null for non-object schemas', () => {
    expect(objectFields({ type: 'string' }, SCHEMAS)).toBeNull();
    expect(objectFields({ type: 'object' }, SCHEMAS)).toBeNull();
    expect(objectFields(ref('Missing'), SCHEMAS)).toBeNull();
  });
});

describe('buildExample', () => {
  it('builds an example object, stopping at recursive references', () => {
    expect(buildExample(ref('Pet'), SCHEMAS)).toEqual({
      name: 'Rex',
      tag: '',
      owner: { email: 'user@example.com' },
      friends: [],
    });
  });

  it('prefers example, default, const and enum values', () => {
    expect(buildExample({ type: 'string', default: 'd' }, {})).toBe('d');
    expect(buildExample({ const: 3 }, {})).toBe(3);
    expect(buildExample({ enum: ['a', 'b'] }, {})).toBe('a');
    expect(buildExample({ oneOf: [{ type: 'boolean' }] }, {})).toBe(false);
    expect(buildExample({ anyOf: [{ type: 'number' }] }, {})).toBe(0);
  });

  it('uses format-specific strings and type defaults', () => {
    const formats = ['date-time', 'date', 'email', 'uuid', 'uri', 'url', 'other'];
    expect(formats.map((f) => buildExample({ type: 'string', format: f }, {}))).toEqual([
      '2024-01-01T00:00:00Z',
      '2024-01-01',
      'user@example.com',
      '00000000-0000-0000-0000-000000000000',
      'https://example.com',
      'https://example.com',
      '',
    ]);
    expect(buildExample({ type: 'integer' }, {})).toBe(0);
    expect(buildExample({ type: 'array', items: { type: 'string' } }, {})).toEqual(['']);
    expect(buildExample({}, {})).toBeNull();
    expect(buildExample(ref('Missing'), {})).toBeNull();
    expect(buildExample({ type: 'object' }, {})).toEqual({});
    expect(buildExample({ type: 'string' }, {}, 7)).toBeUndefined();
  });
});

describe('operation helpers', () => {
  it('picks the first 2xx response, then default', () => {
    const ok = { status: '201' };
    expect(successResponse(operation('a', { responses: [{ status: '400' }, ok, { status: 'default' }] }))).toBe(ok);
    expect(successResponse(operation('a', { responses: [{ status: 'default', description: 'd' }] }))?.description).toBe('d');
    expect(successResponse(operation('a'))).toBeUndefined();
  });

  it('labels operations by summary or method and path', () => {
    expect(operationLabel(operation('a', { summary: 'List pets' }))).toBe('List pets');
    expect(operationLabel(operation('a', { method: 'POST', path: '/pets' }))).toBe('POST /pets');
  });

  it('matches a query against id, path, summary, method and tags', () => {
    const op = operation('listPets', { path: '/pets', summary: 'List all', tags: ['Animals'] });
    expect(operationMatches(op, '')).toBe(true);
    expect(operationMatches(op, '  ANIMALS ')).toBe(true);
    expect(operationMatches(op, 'listpets')).toBe(true);
    expect(operationMatches(op, 'get')).toBe(true);
    expect(operationMatches(op, 'users')).toBe(false);
    expect(operationMatches(operation('x'), 'zzz')).toBe(false);
  });

  it('groups by first tag in order of appearance', () => {
    const ops = [
      operation('a', { tags: ['pets', 'x'] }),
      operation('b'),
      operation('c', { tags: ['users'] }),
      operation('d', { tags: ['pets'] }),
      operation('e', { tags: [] }),
    ];
    expect(groupByTag(ops, 'Other').map(([tag, list]) => [tag, list.map((o) => o.id)])).toEqual([
      ['pets', ['a', 'd']],
      ['Other', ['b', 'e']],
      ['users', ['c']],
    ]);
  });

  it('formats placeholders and keeps unknown ones', () => {
    expect(format('{count} of {total} {missing}', { count: 1, total: '2' })).toBe('1 of 2 {missing}');
  });
});
