import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { Sidebar } from '@/components/navigation';
import { useUIStore } from '@/store/uiStore';
import { useAuthStore } from '@/store/authStore';
import { mockApi, type ApiMock } from '@/test/apiMock';

const Location = () => <div data-testid="location">{useLocation().pathname}</div>;

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Sidebar />
      <Routes>
        <Route path="*" element={<Location />} />
      </Routes>
    </MemoryRouter>,
  );

const path = () => screen.getByTestId('location').textContent;

describe('Sidebar', () => {
  let api: ApiMock;
  beforeEach(() => {
    api = mockApi();
    useUIStore.setState({ language: 'en', isSidebarCollapsed: false });
    useAuthStore.setState({ isAuthenticated: false });
  });
  afterEach(() => api.restore());

  it('navigates to each section and highlights the active one', async () => {
    renderAt('/workflows');
    expect(screen.getByRole('button', { name: 'Workflows List' }).className).toContain('bg-blue-600');
    expect(screen.getByRole('button', { name: 'Dashboard' }).className).not.toContain('bg-blue-600');

    const targets: [string, string][] = [
      ['Dashboard', '/dashboard'],
      ['Workflow Builder', '/builder'],
      ['Executions', '/executions'],
      ['Triggers', '/triggers'],
      ['Monitoring', '/monitoring'],
      ['Resources', '/resources'],
      ['API Clients', '/api-clients'],
      ['Settings', '/settings'],
      ['Workflows List', '/workflows'],
    ];
    for (const [label, target] of targets) {
      await userEvent.click(screen.getByRole('button', { name: label }));
      expect(path()).toBe(target);
      expect(screen.getByRole('button', { name: label }).className).toContain('bg-blue-600');
    }
    expect(screen.queryByText('Sign Out')).not.toBeInTheDocument();
  });

  it('goes home from the logo and treats "/" as exact', async () => {
    const { container } = renderAt('/');
    expect(screen.getByText('MBFlow')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Dashboard' }).className).not.toContain('bg-blue-600');
    await userEvent.click(screen.getByRole('button', { name: 'Triggers' }));
    await userEvent.click(container.querySelector('.cursor-pointer') as HTMLElement);
    expect(path()).toBe('/');
  });

  it('collapses and expands, showing titles when collapsed', async () => {
    renderAt('/dashboard');
    const buttons = screen.getAllByRole('button');
    const toggle = buttons[buttons.length - 1];
    await userEvent.click(toggle);
    expect(useUIStore.getState().isSidebarCollapsed).toBe(true);
    expect(screen.getByTitle('Dashboard')).toBeInTheDocument();
    await userEvent.click(toggle);
    expect(useUIStore.getState().isSidebarCollapsed).toBe(false);
    expect(screen.queryByTitle('Dashboard')).not.toBeInTheDocument();
  });

  it('signs out through the API and redirects to /login', async () => {
    api.on('POST', '/auth/logout', { data: {} });
    useAuthStore.setState({ isAuthenticated: true });
    renderAt('/settings');
    await userEvent.click(screen.getByRole('button', { name: 'Sign Out' }));
    await waitFor(() => expect(path()).toBe('/login'));
    expect(api.calls.some((c) => c.url === '/auth/logout')).toBe(true);
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});
