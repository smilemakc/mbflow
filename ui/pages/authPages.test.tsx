import { beforeEach, describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { LoginPage } from './LoginPage';
import { RegisterPage } from './RegisterPage';
import { useUIStore } from '@/store/uiStore';
import { makeUser, renderPage, signIn } from '@/test/pagesRouter';

beforeEach(() => {
  useUIStore.setState({ language: 'en' });
  signIn(null);
});

describe.each([
  ['LoginPage', LoginPage, '/login', 'Sign In'],
  ['RegisterPage', RegisterPage, '/register', 'Sign Up'],
])('%s', (_name, Page, path, heading) => {
  it('shows the branded form to anonymous users', () => {
    renderPage(<Page />, { at: path });
    expect(screen.getByRole('heading', { name: 'MBFlow' })).toBeInTheDocument();
    expect(screen.getByText('Workflow Orchestration Engine')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: heading })).toBeInTheDocument();
  });

  it('redirects signed-in users home', () => {
    signIn(makeUser());
    renderPage(<Page />, { at: path });
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/);
  });
});
