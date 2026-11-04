import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TransactionHistory } from './TransactionHistory';
import { useUIStore } from '@/store/uiStore';
import { transaction } from '@/test/resourcesFixtures';

const rowFor = (text: string) => screen.getByText(text).closest('tr')!;

describe('TransactionHistory', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('shows the empty state and the counts', async () => {
    const onDeposit = vi.fn();
    render(<TransactionHistory transactions={[]} total={0} onDepositClick={onDeposit} />);
    expect(screen.getByText('No transactions yet')).toBeInTheDocument();
    expect(screen.getByText('Showing 0 of 0 total transactions')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Deposit Funds' }));
    expect(onDeposit).toHaveBeenCalledTimes(1);
  });

  it('renders each transaction with sign, type, status and date', () => {
    render(
      <TransactionHistory
        total={12}
        onDepositClick={vi.fn()}
        transactions={[
          transaction({ id: '1', type: 'deposit', amount: 10, description: 'Top up' }),
          transaction({ id: '2', type: 'charge', amount: 3.5, status: 'pending', description: 'Storage fee' }),
          transaction({ id: '3', type: 'refund', amount: 1, status: 'failed', description: 'Refund A' }),
          // Adjustments credit the balance on the backend.
          transaction({ id: '4', type: 'adjustment', amount: 2, status: 'reversed', description: 'Manual fix' }),
          transaction({ id: '5', type: 'charge', amount: 7, description: '' }),
        ]}
      />,
    );
    expect(screen.getByText('Showing 5 of 12 total transactions')).toBeInTheDocument();
    expect(screen.getAllByRole('row')).toHaveLength(6);

    const deposit = rowFor('Top up');
    expect(within(deposit).getByText('Deposit')).toBeInTheDocument();
    expect(within(deposit).getByText('+10.00 USD').className).toContain('green');
    expect(within(deposit).getByText('Completed').className).toContain('green');
    expect(within(deposit).getByText(/Mar 15, 2026/)).toBeInTheDocument();

    const charge = rowFor('Storage fee');
    expect(within(charge).getByText('Charge')).toBeInTheDocument();
    expect(within(charge).getByText('-3.50 USD').className).toContain('red');
    expect(within(charge).getByText('Pending').className).toContain('yellow');

    const refund = rowFor('Refund A');
    expect(within(refund).getByText('+1.00 USD')).toBeInTheDocument();
    expect(within(refund).getByText('Failed').className).toContain('red');

    const adjustment = rowFor('Manual fix');
    expect(within(adjustment).getByText('adjustment')).toBeInTheDocument();
    expect(within(adjustment).getByText('+2.00 USD')).toBeInTheDocument();
    expect(within(adjustment).getByText('reversed')).toBeInTheDocument();

    expect(within(rowFor('-')).getByText('-7.00 USD')).toBeInTheDocument();
  });
});
