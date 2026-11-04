import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { Permissions, RoleNames } from './auth';
import { PAGES_METADATA, PageType } from './pages';
import { TRIGGER_TYPES } from './triggers';
import * as typesIndex from './index';

const appSource = readFileSync(resolve(__dirname, '../App.tsx'), 'utf8');

describe('auth constants', () => {
  it('use the backend "resource:action" permission strings (go/pkg/models/user.go)', () => {
    expect(Object.values(Permissions)).toEqual([
      'workflow:create',
      'workflow:read',
      'workflow:update',
      'workflow:delete',
      'workflow:execute',
      'execution:read',
      'execution:cancel',
      'execution:retry',
      'trigger:create',
      'trigger:read',
      'trigger:update',
      'trigger:delete',
      'user:manage',
      'role:manage',
      'system:admin',
    ]);
  });

  it('name the built-in roles', () => {
    expect(RoleNames).toEqual({ ADMIN: 'admin', USER: 'user', VIEWER: 'viewer' });
  });
});

describe('PAGES_METADATA', () => {
  it('describes every page type with a route that the app serves', () => {
    expect(Object.keys(PAGES_METADATA).sort()).toEqual(Object.values(PageType).sort());
    Object.entries(PAGES_METADATA).forEach(([key, page]) => {
      expect(page.id).toBe(key);
      expect(page.path).toBe(`/${key}`);
      expect(page.title).not.toBe('');
      expect(page.description).not.toBe('');
      expect(appSource).toContain(`path="${page.path}"`);
    });
  });
});

describe('TRIGGER_TYPES', () => {
  it('lists the trigger types of go/pkg/models/trigger.go', () => {
    expect([...TRIGGER_TYPES].sort()).toEqual(['cron', 'event', 'interval', 'manual', 'webhook']);
  });
});

describe('types index', () => {
  it('re-exports the runtime values of the type modules', () => {
    expect(typesIndex.Permissions).toBe(Permissions);
    expect(typesIndex.TRIGGER_TYPES).toBe(TRIGGER_TYPES);
    expect(typesIndex.DEFAULT_NODE_CONFIGS).toBeDefined();
    expect(typesIndex.isValidNodeType('http')).toBe(true);
    expect(typesIndex.isValidNodeType('nope')).toBe(false);
  });
});
