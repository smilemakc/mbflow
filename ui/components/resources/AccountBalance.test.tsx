import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AccountBalance } from './AccountBalance';
import { useUIStore } from '@/store/uiStore';
import { account } from '@/test/resourcesFixtures';

describe('AccountBalance', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('shows the balance with two decimals, currency and a capitalised active status', () => {
    render(<AccountBalance account={account({ balance: 125.5, currency: 'EUR' })} />);
    expect(screen.getByText('Account Balance')).toBeInTheDocument();
    expect(screen.getByText(/125\.50 EUR/)).toBeInTheDocument();
    const badge = screen.getByText('Active');
    expect(badge.className).toContain('green');
  });

  it('renders a neutral badge for non-active accounts', () => {
    render(<AccountBalance account={account({ balance: 0, status: 'suspended' })} />);
    expect(screen.getByText(/0\.00 USD/)).toBeInTheDocument();
    const badge = screen.getByText('Suspended');
    expect(badge.className).toContain('gray');
    expect(badge.className).not.toContain('green');
  });

  it('uses the Russian label when the language is ru', () => {
    useUIStore.setState({ language: 'ru' });
    render(<AccountBalance account={account()} />);
    expect(screen.queryByText('Account Balance')).not.toBeInTheDocument();
  });
});
