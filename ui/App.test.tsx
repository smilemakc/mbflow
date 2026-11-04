import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import App from './App';
import { useAuthStore } from '@/store/authStore';
import { useUIStore } from '@/store/uiStore';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { makeUser } from '@/test/pagesRouter';
import { TOKEN_KEY } from '@/lib/authStorage';

// Pages are lazy-loaded; stand-ins keep this test about routing and guards.
const stub = (label: string) => () => <div>{label} page</div>;
vi.mock('@/layouts/BuilderLayout', () => ({ BuilderLayout: stub('builder') }));
vi.mock('@/pages/DashboardPage', () => ({ DashboardPage: stub('dashboard') }));
vi.mock('@/pages/ExecutionDetailPage', () => ({ ExecutionDetailPage: stub('execution detail') }));
vi.mock('@/pages/ExecutionsPage', () => ({ ExecutionsPage: stub('executions') }));
vi.mock('@/pages/MonitoringPage', () => ({ MonitoringPage: stub('monitoring') }));
vi.mock('@/pages/ResourcesPage', () => ({ ResourcesPage: stub('resources') }));
vi.mock('@/pages/ApiClientsPage', () => ({ ApiClientsPage: stub('api clients') }));
vi.mock('@/pages/SettingsPage', () => ({ SettingsPage: stub('settings') }));
vi.mock('@/pages/TriggersPage', () => ({ TriggersPage: stub('triggers') }));
vi.mock('@/pages/WorkflowsPage', () => ({ WorkflowsPage: stub('workflows') }));
vi.mock('@/pages/LoginPage', () => ({ LoginPage: stub('login') }));
vi.mock('@/pages/RegisterPage', () => ({ RegisterPage: stub('register') }));
vi.mock('@/pages/UsersPage', () => ({ UsersPage: stub('users') }));
vi.mock('@/pages/RentalKeysAdminPage', () => ({ RentalKeysAdminPage: stub('rental keys admin') }));

let api: ApiMock;

const go = (hash: string) => {
  window.location.hash = hash;
};

const signInWithToken = (user = makeUser()) => {
  localStorage.setItem(TOKEN_KEY, 'tok');
  api.on('GET', '/auth/me', { data: { user } });
};

beforeEach(() => {
  api = mockApi();
  useUIStore.setState({ language: 'en', theme: 'light', activeModal: null });
  useAuthStore.setState({ user: null, isAuthenticated: false, isInitialized: false, isLoading: false, error: null });
});

afterEach(() => {
  api.restore();
  go('');
});

describe('App', () => {
  it('shows a loading screen until auth is checked, then the route', async () => {
    let release!: () => void;
    localStorage.setItem(TOKEN_KEY, 'tok');
    api.on('GET', '/auth/me', () => new Promise((resolve) => (release = () => resolve({ data: { user: makeUser() } }))));
    go('#/workflows');
    render(<App />);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
    await waitFor(() => expect(release).toBeTypeOf('function'));
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    release();
    expect(await screen.findByText('workflows page')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'Workflows' })).toBeInTheDocument();
  });

  it('applies the theme to the document root', async () => {
    useUIStore.setState({ theme: 'dark' });
    go('#/login');
    render(<App />);
    await screen.findByText('login page');
    expect(document.documentElement).toHaveClass('dark');
    expect(document.documentElement).not.toHaveClass('light');
  });

  it('sends anonymous users to the login page from protected routes', async () => {
    go('#/executions');
    render(<App />);
    expect(await screen.findByText('login page')).toBeInTheDocument();
    expect(window.location.hash).toBe('#/login');
  });

  it('serves public auth routes', async () => {
    go('#/register');
    render(<App />);
    expect(await screen.findByText('register page')).toBeInTheDocument();
  });

  it.each([
    ['#/dashboard', 'dashboard', 'Dashboard'],
    ['#/executions', 'executions', 'Execution History'],
    ['#/executions/abc', 'execution detail', 'Execution Details'],
    ['#/triggers', 'triggers', 'Triggers'],
    ['#/monitoring', 'monitoring', 'System Monitoring'],
    ['#/resources', 'resources', 'Resources'],
    ['#/api-clients', 'api clients', 'API Clients'],
    ['#/settings', 'settings', 'Settings'],
  ])('renders %s inside the page layout for signed-in users', async (hash, page, title) => {
    signInWithToken();
    go(hash);
    render(<App />);
    expect(await screen.findByText(`${page} page`)).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: title })).toBeInTheDocument();
  });

  it.each(['#/', '#/does-not-exist', '#/builder/wf-1'])('routes %s to the builder', async (hash) => {
    signInWithToken();
    go(hash);
    render(<App />);
    expect(await screen.findByText('builder page')).toBeInTheDocument();
    expect(window.location.hash).toMatch(/^#\/builder/);
  });

  it('keeps admin pages for admins only', async () => {
    signInWithToken(makeUser({ is_admin: false }));
    go('#/admin/users');
    const first = render(<App />);
    expect(await screen.findByText('builder page')).toBeInTheDocument();
    first.unmount();

    api.restore();
    api = mockApi();
    signInWithToken(makeUser({ is_admin: true }));
    useAuthStore.setState({ isInitialized: false });
    go('#/admin/users');
    const second = render(<App />);
    expect(await screen.findByText('users page')).toBeInTheDocument();
    second.unmount();

    useAuthStore.setState({ isInitialized: false });
    go('#/admin/rental-keys');
    render(<App />);
    expect(await screen.findByText('rental keys admin page')).toBeInTheDocument();
  });
});
