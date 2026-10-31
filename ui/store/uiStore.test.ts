import { beforeEach, describe, expect, it } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useUIStore } from './uiStore';
import { useTranslation, useTranslations } from './translations';

describe('uiStore', () => {
  beforeEach(() => {
    useUIStore.setState({
      theme: 'light',
      language: 'en',
      isSidebarCollapsed: false,
      isFullscreen: false,
      activeModal: null,
      isNodeLibraryOpen: false,
    });
  });

  it('toggles and sets UI state', () => {
    const s = () => useUIStore.getState();
    s().toggleTheme();
    expect(s().theme).toBe('dark');
    s().toggleTheme();
    expect(s().theme).toBe('light');
    s().setTheme('dark');
    expect(s().theme).toBe('dark');

    s().toggleLanguage();
    expect(s().language).toBe('ru');
    s().toggleLanguage();
    expect(s().language).toBe('en');
    s().setLanguage('ru');
    expect(s().language).toBe('ru');

    s().toggleSidebar();
    s().toggleFullscreen();
    s().toggleNodeLibrary();
    s().setActiveModal('templates');
    expect(s()).toMatchObject({ isSidebarCollapsed: true, isFullscreen: true, isNodeLibraryOpen: true, activeModal: 'templates' });
  });

  it('persists preferences but not transient state', () => {
    useUIStore.getState().setTheme('dark');
    useUIStore.getState().toggleFullscreen();
    const stored = JSON.parse(localStorage.getItem('ui-storage') || '{}');
    expect(stored.state).toEqual({ theme: 'dark', language: 'en', isSidebarCollapsed: false, isNodeLibraryOpen: false });
  });
});

describe('translations', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('returns the dictionary for the current language', () => {
    const { result } = renderHook(() => useTranslation());
    expect(result.current.triggers.cron).toBe('Cron schedule');
    act(() => useUIStore.getState().setLanguage('ru'));
    expect(result.current.triggers.cron).toBe('Cron-расписание');
  });

  it('falls back to English for an unknown language', () => {
    useUIStore.setState({ language: 'de' as never });
    const { result } = renderHook(() => useTranslation());
    expect(result.current.common.save).toBe('Save');
  });

  it('resolves dotted keys with English and explicit fallbacks', () => {
    useUIStore.setState({ language: 'ru' });
    const { result } = renderHook(() => useTranslations());
    const t = result.current;
    expect(t('common.save')).toBe('Сохранить');
    expect(t('missing.key', 'Fallback')).toBe('Fallback');
    expect(t('missing.key')).toBe('missing.key');
    expect(t('common')).toBe('common');
  });

  it('uses the English value when the current language lacks a key', () => {
    useUIStore.setState({ language: 'xx' as never });
    const { result } = renderHook(() => useTranslations());
    expect(result.current('common.save')).toBe('Save');
  });

  it('keeps English and Russian dictionaries in sync for the sections touched here', () => {
    const keys = (obj: unknown, prefix = ''): string[] =>
      obj && typeof obj === 'object'
        ? Object.entries(obj as Record<string, unknown>).flatMap(([k, v]) => keys(v, `${prefix}${k}.`))
        : [prefix.slice(0, -1)];
    const { result } = renderHook(() => useTranslation());
    const en = result.current;
    act(() => useUIStore.getState().setLanguage('ru'));
    const ru = result.current;
    for (const section of ['triggers', 'system', 'settings', 'dashboard'] as const) {
      expect(keys(ru[section]).sort()).toEqual(keys(en[section]).sort());
    }
  });
});
