import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SettingsPage } from './SettingsPage';
import { authService } from '@/services/authService';
import { useAuthStore } from '@/store/authStore';
import { useUIStore } from '@/store/uiStore';
import type { User } from '@/types/auth';

const me: User = {
  id: 'u1',
  email: 'ada@example.com',
  username: 'ada',
  full_name: 'Ada Lovelace',
  is_active: true,
  is_admin: true,
  roles: ['admin', 'user'],
  created_at: '2025-01-02T03:04:05Z',
  updated_at: '2025-01-02T03:04:05Z',
  last_login_at: '2026-10-30T08:00:00Z',
};

describe('SettingsPage', () => {
  beforeEach(() => {
    useUIStore.setState({ language: 'en', theme: 'light' });
    useAuthStore.setState({ user: null, isLoading: false });
  });

  it('shows the profile returned by /auth/me instead of demo data', async () => {
    const getMe = vi.spyOn(authService, 'getCurrentUser').mockResolvedValue(me);
    render(<SettingsPage />);

    expect(await screen.findByText('Ada Lovelace')).toBeInTheDocument();
    expect(getMe).toHaveBeenCalledTimes(1);
    expect(screen.getByText('ada@example.com')).toBeInTheDocument();
    expect(screen.getByText('Administrator')).toBeInTheDocument();
    expect(screen.getByText('admin, user')).toBeInTheDocument();
    expect(screen.queryByText(/Alex|Morgan|Weekly Digest|Notifications|Billing/)).toBeNull();
    expect(screen.getByText('Last sign-in').nextElementSibling?.textContent).not.toBe('-');
  });

  it('falls back gracefully for a sparse profile and failed reloads', async () => {
    const user = userEvent.setup();
    const getMe = vi
      .spyOn(authService, 'getCurrentUser')
      .mockResolvedValueOnce({ ...me, full_name: undefined, username: '', is_admin: false, roles: [], last_login_at: undefined })
      .mockRejectedValueOnce(new Error('offline'));
    render(<SettingsPage />);

    expect(await screen.findByText('No roles assigned')).toBeInTheDocument();
    expect(screen.getByText('User')).toBeInTheDocument();
    expect(screen.getByText('Last sign-in').nextElementSibling).toHaveTextContent('-');

    // A failed reload keeps the last known profile.
    await user.click(screen.getByRole('button', { name: 'Reload' }));
    await waitFor(() => expect(getMe).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('reports when the profile cannot be loaded', async () => {
    vi.spyOn(authService, 'getCurrentUser').mockRejectedValue(new Error('offline'));
    render(<SettingsPage />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load your profile.');
  });

  it('validates and submits a password change', async () => {
    vi.spyOn(authService, 'getCurrentUser').mockResolvedValue(me);
    const changePassword = vi.fn().mockResolvedValueOnce(false).mockResolvedValueOnce(true);
    useAuthStore.setState({ changePassword });
    const user = userEvent.setup();
    render(<SettingsPage />);

    await user.click(screen.getByRole('button', { name: 'Security' }));
    const submit = () => user.click(screen.getByRole('button', { name: 'Change password' }));

    await submit();
    expect(screen.getByRole('alert')).toHaveTextContent('Enter your current password');

    await user.type(screen.getByLabelText('Current password'), 'old-password');
    await user.type(screen.getByLabelText('New password'), 'short');
    await submit();
    expect(screen.getByRole('alert')).toHaveTextContent('Password must be at least 8 characters');

    await user.clear(screen.getByLabelText('New password'));
    await user.type(screen.getByLabelText('New password'), 'long-enough');
    await user.type(screen.getByLabelText('Confirm new password'), 'different');
    await submit();
    expect(screen.getByRole('alert')).toHaveTextContent('Passwords do not match');

    await user.clear(screen.getByLabelText('Confirm new password'));
    await user.type(screen.getByLabelText('Confirm new password'), 'long-enough');
    await submit();
    expect(changePassword).toHaveBeenCalledWith({ old_password: 'old-password', new_password: 'long-enough' });
    expect(screen.queryByRole('alert')).toBeNull();
    // Failed change keeps the inputs.
    expect(screen.getByLabelText('Current password')).toHaveValue('old-password');

    await submit();
    await waitFor(() => expect(screen.getByLabelText('Current password')).toHaveValue(''));
  });

  it('switches theme and language', async () => {
    vi.spyOn(authService, 'getCurrentUser').mockResolvedValue(me);
    const user = userEvent.setup();
    render(<SettingsPage />);

    await user.click(screen.getByRole('button', { name: 'Appearance' }));
    expect(screen.getByRole('button', { name: 'Appearance' })).toHaveAttribute('aria-current', 'page');
    await user.click(screen.getByRole('radio', { name: 'Dark' }));
    expect(useUIStore.getState().theme).toBe('dark');
    expect(screen.getByRole('radio', { name: 'Dark' })).toHaveAttribute('aria-checked', 'true');

    await user.click(screen.getByRole('radio', { name: 'Русский' }));
    expect(useUIStore.getState().language).toBe('ru');
    expect(screen.getByRole('heading', { name: 'Настройки' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Профиль' }));
    expect(await screen.findByText('Ada Lovelace')).toBeInTheDocument();
  });
});
