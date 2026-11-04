import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DepositModal } from './DepositModal';
import { useUIStore } from '@/store/uiStore';

const deferred = <T,>() => {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
};

describe('DepositModal', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('renders nothing when closed', () => {
    render(<DepositModal isOpen={false} onClose={vi.fn()} onSubmit={vi.fn()} />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('only enables Deposit for a positive amount', async () => {
    const user = userEvent.setup();
    render(<DepositModal isOpen onClose={vi.fn()} onSubmit={vi.fn()} />);
    const deposit = screen.getByRole('button', { name: 'Deposit' });
    const input = screen.getByPlaceholderText('0.00');
    expect(deposit).toBeDisabled();

    await user.type(input, '0');
    expect(deposit).toBeDisabled();
    await user.clear(input);
    await user.type(input, '-5');
    expect(deposit).toBeDisabled();
    await user.clear(input);
    await user.type(input, '12.5');
    expect(deposit).toBeEnabled();
    expect(screen.getByText('Enter the amount you want to add to your account balance')).toBeInTheDocument();
  });

  it('submits the parsed amount, shows loading, then resets and closes on success', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const d = deferred<boolean>();
    const onSubmit = vi.fn(() => d.promise);
    render(<DepositModal isOpen onClose={onClose} onSubmit={onSubmit} />);

    await user.type(screen.getByPlaceholderText('0.00'), '42.75');
    await user.click(screen.getByRole('button', { name: 'Deposit' }));
    expect(onSubmit).toHaveBeenCalledWith(42.75);

    // While loading, cancel / escape are ignored.
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    await user.keyboard('{Escape}');
    expect(onClose).not.toHaveBeenCalled();

    d.resolve(true);
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(screen.getByPlaceholderText('0.00')).toHaveValue(null);
  });

  it('keeps the modal and amount when the deposit fails', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSubmit = vi.fn().mockResolvedValue(false);
    render(<DepositModal isOpen onClose={onClose} onSubmit={onSubmit} />);

    await user.type(screen.getByPlaceholderText('0.00'), '5');
    await user.click(screen.getByRole('button', { name: 'Deposit' }));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Deposit' })).toBeEnabled());
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByPlaceholderText('0.00')).toHaveValue(5);
  });

  it('clears the amount and closes on cancel', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<DepositModal isOpen onClose={onClose} onSubmit={vi.fn()} />);
    await user.type(screen.getByPlaceholderText('0.00'), '9');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.getByPlaceholderText('0.00')).toHaveValue(null);
  });
});
