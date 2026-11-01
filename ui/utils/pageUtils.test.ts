import { describe, expect, it } from 'vitest';
import { PageType } from '@/types/pages';
import {
  getAllPages,
  getPageMetadata,
  getPagePath,
  getPageTitle,
  getPageTypeFromPath,
  isValidPageType,
} from './pageUtils';

describe('pageUtils', () => {
  it('looks up metadata', () => {
    expect(getPageMetadata(PageType.MONITORING).path).toBe('/monitoring');
    expect(getPagePath(PageType.SETTINGS)).toBe('/settings');
    expect(getPageTitle(PageType.DASHBOARD)).toBe('Dashboard');
    expect(getAllPages()).toHaveLength(Object.values(PageType).length);
  });

  it('falls back for unknown pages', () => {
    expect(getPagePath('nope' as PageType)).toBe('/');
    expect(getPageTitle('nope' as PageType)).toBe('Page');
  });

  it('validates page types', () => {
    expect(isValidPageType('resources')).toBe(true);
    expect(isValidPageType('builder')).toBe(false);
  });

  it('resolves paths with or without a leading slash', () => {
    expect(getPageTypeFromPath('/dashboard')).toBe(PageType.DASHBOARD);
    expect(getPageTypeFromPath('resources')).toBe(PageType.RESOURCES);
    expect(getPageTypeFromPath('/unknown')).toBeNull();
  });
});
