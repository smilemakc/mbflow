import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RotateKeyModal } from './RotateKeyModal';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { adminRentalKey } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';

const input = () => screen.getByPlaceholderText('Enter the new API key');
const submit = () => screen.getByRole('button', { name: 'Rotate Key' });

describe('RotateKeyModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi();
  });

  afterEach(() => api.restore());

  const renderModal = () => {
    const onClose = vi.fn();
    const onRotated = vi.fn();
    render(
      <RotateKeyModal
        isOpen
        onClose={onClose}
        onRotated={onRotated}
        rentalKey={adminRentalKey('k1', { name: 'Gemini', provider: 'google_ai' })}
      />,
    );
    return { onClose, onRotated };
  };

  it('shows the key being rotated and the warning', () => {
    renderModal();
    expect(screen.getByText('Gemini')).toBeInTheDocument();
    expect(screen.getByText('Google AI')).toBeInTheDocument();
    expect(screen.getByText(/immediately invalidate the old key/)).toBeInTheDocument();
  });

  it('rejects a blank key without calling the API', async () => {
    renderModal();
    await userEvent.type(input(), '   ');
    await userEvent.click(submit());
    expect(await screen.findByText('API key is required')).toBeInTheDocument();
    expect(api.calls).toHaveLength(0);
  });

  it('posts the new key and closes', async () => {
    api.on('POST', '/admin/rental-keys/k1/rotate-key', { data: { message: 'API key rotated successfully' } });
    const { onClose, onRotated } = renderModal();
    await userEvent.type(input(), 'sk-new');
    await userEvent.click(submit());
    await waitFor(() => expect(onRotated).toHaveBeenCalled());
    expect(onClose).toHaveBeenCalled();
    expect(api.last()!.data).toEqual({ new_api_key: 'sk-new' });
    expect(input()).toHaveValue('');
  });

  it('shows the API error, and resets the form when cancelled', async () => {
    api.on('POST', '/admin/rental-keys/k1/rotate-key', { status: 500, data: { message: 'failed to rotate' } });
    const { onClose, onRotated } = renderModal();
    await userEvent.type(input(), 'sk-new');
    await userEvent.click(submit());
    expect(await screen.findByText('failed to rotate')).toBeInTheDocument();
    expect(onRotated).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(input()).toHaveValue('');
    expect(screen.queryByText('failed to rotate')).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Close modal' }));
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
