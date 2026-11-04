/**
 * Test helper for page-level tests: renders an element inside a MemoryRouter
 * at a given path and exposes the current location for navigation assertions.
 */
import React from 'react';
import { render } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import type { User } from '@/types/auth';

export const LocationProbe: React.FC = () => {
  const location = useLocation();
  return (
    <div data-testid="location" data-state={JSON.stringify(location.state ?? null)}>
      {location.pathname}
    </div>
  );
};

interface RenderOptions {
  /** Route pattern the element is mounted on (default: the initial path). */
  path?: string;
  /** Initial URL (default: '/'). */
  at?: string;
  /** Initial history state. */
  state?: unknown;
}

/** Mounts `ui` at `path`; any other location renders a probe with the pathname. */
export const renderPage = (ui: React.ReactElement, { path, at = '/', state }: RenderOptions = {}) =>
  render(
    <MemoryRouter initialEntries={[{ pathname: at, state }]}>
      <Routes>
        <Route path={path ?? at} element={ui} />
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  );

export const makeUser = (overrides: Partial<User> = {}): User => ({
  id: 'u1',
  email: 'ada@example.com',
  username: 'ada',
  full_name: 'Ada Lovelace',
  is_active: true,
  is_admin: false,
  roles: ['user'],
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
  ...overrides,
});

/** Puts the auth store into a signed-in (or signed-out) state. */
export const signIn = (user: User | null) =>
  useAuthStore.setState({
    user,
    isAuthenticated: !!user,
    isInitialized: true,
    isLoading: false,
    error: null,
  });
