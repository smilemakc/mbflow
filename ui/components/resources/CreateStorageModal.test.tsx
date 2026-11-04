import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateStorageModal } from './CreateStorageModal';
import { useUIStore } from '@/store/uiStore';

describe('CreateStorageModal', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  const nameInput = () => screen.getByPlaceholderText('e.g., Production Storage');
  const descInput = () => screen.getByPlaceholderText('Optional description...');

  it('requires a non-blank name', async () => {
    const user = userEvent.setup();
    render(<CreateStorageModal isOpen onClose={vi.fn()} onSubmit={vi.fn()} />);
    expect(screen.getByRole('heading', { name: 'Create File Storage' })).toBeInTheDocument();
    const create = screen.getByRole('button', { name: 'Create' });
    expect(create).toBeDisabled();
    await user.type(nameInput(), '   ');
    expect(create).toBeDisabled();
    await user.type(nameInput(), 'x');
    expect(create).toBeEnabled();
  });

  it('submits trimmed values and resets/closes on success', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    let resolve!: (v: boolean) => void;
    const onSubmit = vi.fn(() => new Promise<boolean>((r) => (resolve = r)));
    render(<CreateStorageModal isOpen onClose={onClose} onSubmit={onSubmit} />);

    await user.type(nameInput(), '  Backups ');
    await user.type(descInput(), ' nightly dumps  ');
    await user.click(screen.getByRole('button', { name: 'Create' }));
    expect(onSubmit).toHaveBeenCalledWith('Backups', 'nightly dumps');

    // Closing is blocked while the request is in flight.
    await user.click(screen.getByRole('button', { name: 'Close modal' }));
    expect(onClose).not.toHaveBeenCalled();

    resolve(true);
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(nameInput()).toHaveValue('');
    expect(descInput()).toHaveValue('');
  });

  it('keeps the form populated when creation fails', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<CreateStorageModal isOpen onClose={onClose} onSubmit={vi.fn().mockResolvedValue(false)} />);
    await user.type(nameInput(), 'Logs');
    await user.click(screen.getByRole('button', { name: 'Create' }));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Create' })).toBeEnabled());
    expect(onClose).not.toHaveBeenCalled();
    expect(nameInput()).toHaveValue('Logs');
  });

  it('cancel clears the fields and closes', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<CreateStorageModal isOpen onClose={onClose} onSubmit={vi.fn()} />);
    await user.type(nameInput(), 'Tmp');
    await user.type(descInput(), 'd');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(nameInput()).toHaveValue('');
    expect(descInput()).toHaveValue('');
  });
});
