import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { LoginForm, ProtectedRoute, RegisterForm, UserMenu } from '@/components/auth';
import { useAuthStore } from '@/store/authStore';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { LocationProbe, makeUser, renderPage, signIn } from '@/test/pagesRouter';
import { TOKEN_KEY, REFRESH_TOKEN_KEY } from '@/lib/authStorage';

const toasts = () => useToastStore.getState().toasts;

let api: ApiMock;

beforeEach(() => {
  useUIStore.setState({ language: 'en' });
  useToastStore.getState().clearAll();
  signIn(null);
  api = mockApi();
});

afterEach(() => api.restore());

describe('LoginForm', () => {
  const fill = async (user: ReturnType<typeof userEvent.setup>, email = 'ada@example.com', password = 'secret123') => {
    await user.type(screen.getByLabelText('Email'), email);
    await user.type(screen.getByLabelText('Password'), password);
    await user.click(screen.getByRole('button', { name: 'Sign In' }));
  };

  it('signs in, stores tokens and returns to the page the user came from', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/login', {
      data: { access_token: 'tok', refresh_token: 'ref', user: makeUser() },
    });
    renderPage(<LoginForm />, { at: '/login', state: { from: { pathname: '/triggers' } } });

    await fill(user);

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/triggers'));
    expect(api.last()).toMatchObject({ method: 'POST', url: '/auth/login', data: { email: 'ada@example.com', password: 'secret123' } });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('tok');
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBe('ref');
    expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: true, user: { email: 'ada@example.com' } });
  });

  it('goes to the root when there is no previous location', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/login', { data: { access_token: 'tok', user: makeUser() } });
    renderPage(<LoginForm />, { at: '/login' });
    await fill(user);
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/));
  });

  it('shows the API error and stays on the form', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/login', { status: 400, data: { code: 'INVALID_CREDENTIALS', message: 'invalid email or password' } });
    renderPage(<LoginForm />, { at: '/login' });

    await fill(user, 'ada@example.com', 'wrong');

    expect(await screen.findByText('invalid email or password')).toBeInTheDocument();
    expect(screen.queryByTestId('location')).toBeNull();
    expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Login Failed' });
  });

  it('shows the loading label while signing in and links to registration', async () => {
    useAuthStore.setState({ isLoading: true });
    const user = userEvent.setup();
    renderPage(<LoginForm />, { at: '/login' });
    expect(screen.getByRole('button', { name: /Signing in/ })).toBeDisabled();

    useAuthStore.setState({ isLoading: false });
    await user.click(await screen.findByRole('link', { name: 'Sign Up' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/register');
  });
});

describe('RegisterForm', () => {
  const fill = async (
    user: ReturnType<typeof userEvent.setup>,
    { email = 'new@example.com', name = '', password = 'Password1', confirm = password } = {},
  ) => {
    await user.type(screen.getByLabelText(/Email/), email);
    if (name) await user.type(screen.getByLabelText('Full Name'), name);
    await user.type(screen.getByLabelText(/^Password/), password);
    await user.type(screen.getByLabelText(/Confirm Password/), confirm);
    await user.click(screen.getByRole('button', { name: 'Create Account' }));
  };

  it('validates matching passwords and minimum length before calling the API', async () => {
    const user = userEvent.setup();
    renderPage(<RegisterForm />, { at: '/register' });

    await fill(user, { password: 'Password1', confirm: 'Password2' });
    expect(screen.getByText('Passwords do not match')).toBeInTheDocument();

    // Editing a field clears the validation message.
    await user.type(screen.getByLabelText(/Confirm Password/), 'x');
    expect(screen.queryByText('Passwords do not match')).toBeNull();

    await user.clear(screen.getByLabelText(/^Password/));
    await user.clear(screen.getByLabelText(/Confirm Password/));
    await user.type(screen.getByLabelText(/^Password/), 'short');
    await user.type(screen.getByLabelText(/Confirm Password/), 'short');
    await user.click(screen.getByRole('button', { name: 'Create Account' }));
    expect(screen.getByText('Password must be at least 8 characters')).toBeInTheDocument();
    expect(api.calls).toHaveLength(0);
  });

  it('registers the user with the full name and goes home', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/register', { data: { access_token: 'tok', user: makeUser({ email: 'new@example.com' }) } });
    renderPage(<RegisterForm />, { at: '/register' });

    await fill(user, { name: 'New Person' });

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/));
    expect(api.last()?.data).toEqual({ email: 'new@example.com', password: 'Password1', full_name: 'New Person' });
    expect(toasts().at(-1)).toMatchObject({ type: 'success', title: 'Registration successful!' });
  });

  it('omits an empty full name and shows server errors', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/register', { status: 409, data: { code: 'EMAIL_ALREADY_TAKEN', message: 'email already taken' } });
    renderPage(<RegisterForm />, { at: '/register' });

    await fill(user);

    expect(await screen.findByText('email already taken')).toBeInTheDocument();
    expect(api.last()?.data).toEqual({ email: 'new@example.com', password: 'Password1' });
    expect(toasts().at(-1)).toMatchObject({ title: 'Registration Failed', message: 'This email is already registered' });
  });

  it('shows the loading label and links back to sign in', async () => {
    useAuthStore.setState({ isLoading: true });
    const user = userEvent.setup();
    renderPage(<RegisterForm />, { at: '/register' });
    expect(screen.getByRole('button', { name: /Creating account/ })).toBeDisabled();
    useAuthStore.setState({ isLoading: false });
    await user.click(await screen.findByRole('link', { name: 'Sign In' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/login');
  });
});

describe('ProtectedRoute', () => {
  const renderGuarded = (props: { requireAdmin?: boolean; requiredRole?: string } = {}) =>
    render(
      <MemoryRouter initialEntries={['/secret']}>
        <Routes>
          <Route
            path="/secret"
            element={
              <ProtectedRoute {...props}>
                <p>secret content</p>
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

  it('shows a spinner until auth is initialized or while loading', () => {
    useAuthStore.setState({ isInitialized: false });
    const { unmount } = renderGuarded();
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    unmount();

    signIn(makeUser());
    useAuthStore.setState({ isLoading: true });
    renderGuarded();
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('redirects anonymous users to /login remembering where they came from', () => {
    renderGuarded();
    const probe = screen.getByTestId('location');
    expect(probe).toHaveTextContent('/login');
    expect(JSON.parse(probe.dataset.state!)).toMatchObject({ from: { pathname: '/secret' } });
  });

  it('renders children for an authenticated user', () => {
    signIn(makeUser());
    renderGuarded();
    expect(screen.getByText('secret content')).toBeInTheDocument();
  });

  it('enforces admin and role requirements', () => {
    signIn(makeUser({ roles: ['viewer'] }));
    const first = renderGuarded({ requireAdmin: true });
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/);
    first.unmount();

    const second = renderGuarded({ requiredRole: 'editor' });
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/);
    second.unmount();

    renderGuarded({ requiredRole: 'Viewer' });
    expect(screen.getByText('secret content')).toBeInTheDocument();
  });

  it('lets admins through admin-only routes', () => {
    signIn(makeUser({ is_admin: true }));
    renderGuarded({ requireAdmin: true });
    expect(screen.getByText('secret content')).toBeInTheDocument();
  });
});

describe('UserMenu', () => {
  it('offers sign in / sign up links when signed out', async () => {
    const user = userEvent.setup();
    renderPage(<UserMenu />);
    expect(screen.getByRole('link', { name: 'Sign Up' })).toHaveAttribute('href', '/register');
    await user.click(screen.getByRole('link', { name: 'Sign In' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/login');
  });

  it('shows initials and role, and opens/closes the dropdown', async () => {
    const user = userEvent.setup();
    signIn(makeUser({ full_name: 'ada byron lovelace', roles: ['editor'] }));
    render(
      <MemoryRouter>
        <div>
          <span>outside</span>
          <UserMenu />
        </div>
      </MemoryRouter>,
    );

    expect(screen.getByText('AB')).toBeInTheDocument();
    expect(screen.getByText('editor')).toBeInTheDocument();
    expect(screen.queryByText('ada@example.com')).toBeNull();

    await user.click(screen.getByRole('button', { name: /ada byron lovelace/ }));
    expect(screen.getByText('ada@example.com')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings');
    expect(screen.queryByRole('link', { name: 'User Management' })).toBeNull();

    // Clicking inside keeps it open; clicking outside closes it.
    await user.click(screen.getByText('ada@example.com'));
    expect(screen.getByText('ada@example.com')).toBeInTheDocument();
    await user.click(screen.getByText('outside'));
    expect(screen.queryByText('ada@example.com')).toBeNull();

    // A menu link closes the dropdown too.
    await user.click(screen.getByRole('button', { name: /ada byron lovelace/ }));
    await user.click(screen.getByRole('link', { name: 'Settings' }));
    expect(screen.queryByText('ada@example.com')).toBeNull();
  });

  it('falls back to the username and default role label', () => {
    signIn(makeUser({ full_name: undefined, username: 'grace', roles: [] }));
    renderPage(<UserMenu />);
    expect(screen.getByText('GR')).toBeInTheDocument();
    expect(screen.getByText('grace')).toBeInTheDocument();
    expect(screen.getByText('User')).toBeInTheDocument();
  });

  it('shows admin links and logs out', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/logout', { data: {} });
    localStorage.setItem(TOKEN_KEY, 'tok');
    signIn(makeUser({ is_admin: true }));
    renderPage(<UserMenu />);

    expect(screen.getByText('Admin')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Ada Lovelace/ }));
    await user.click(screen.getByRole('link', { name: 'User Management' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/admin/users');
  });

  it('signs out through the API and goes to the login page', async () => {
    const user = userEvent.setup();
    api.on('POST', '/auth/logout', { data: {} });
    localStorage.setItem(TOKEN_KEY, 'tok');
    signIn(makeUser({ is_admin: true }));
    renderPage(<UserMenu />);

    await user.click(screen.getByRole('button', { name: /Ada Lovelace/ }));
    expect(screen.getByRole('link', { name: 'Rental Keys' })).toHaveAttribute('href', '/admin/rental-keys');
    await user.click(screen.getByRole('button', { name: 'Sign Out' }));

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/login'));
    expect(api.calls.map((c) => `${c.method} ${c.url}`)).toContain('POST /auth/logout');
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it('closes on a link click for admin entries', async () => {
    const user = userEvent.setup();
    signIn(makeUser({ is_admin: true }));
    render(
      <MemoryRouter>
        <UserMenu />
      </MemoryRouter>,
    );
    await user.click(screen.getByRole('button', { name: /Ada Lovelace/ }));
    await user.click(screen.getByRole('link', { name: 'Rental Keys' }));
    expect(screen.queryByRole('link', { name: 'Rental Keys' })).toBeNull();
  });
});
