import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RentalKeyUsageModal } from './RentalKeyUsageModal';
import { useUIStore } from '@/store/uiStore';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { rentalKey, usageRecord } from '@/test/resourcesFixtures';

const page = (start: number, count: number) =>
  Array.from({ length: count }, (_, i) => usageRecord({ id: `u${start + i}`, model: `model-${start + i}` }));

describe('RentalKeyUsageModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('shows the multimodal summary and usage records from GET /rental-keys/:id/usage', async () => {
    api.on('GET', /^\/rental-keys\/rk-1\/usage/, {
      data: {
        usage: [
          usageRecord(),
          usageRecord({ id: 'u2', model: 'gpt-4o-mini', status: 'error', response_time_ms: 0, estimated_cost: 0 }),
        ],
        limit: 20,
        offset: 0,
      },
    });
    render(<RentalKeyUsageModal isOpen onClose={vi.fn()} rentalKey={rentalKey()} />);

    expect(screen.getByText('Usage History')).toBeInTheDocument();
    expect(screen.getByText('Team GPT key - OpenAI')).toBeInTheDocument();
    // Summary: text = 1000+500, image = 2000, audio = 30, video = 4M
    expect(screen.getByText('Text Tokens').parentElement!.nextElementSibling).toHaveTextContent('1.50K');
    expect(screen.getByText('Image Tokens').parentElement!.nextElementSibling).toHaveTextContent('2.00K');
    expect(screen.getByText('Audio Tokens').parentElement!.nextElementSibling).toHaveTextContent('30');
    expect(screen.getByText('Video Tokens').parentElement!.nextElementSibling).toHaveTextContent('4.00M');

    expect(await screen.findByText('gpt-4o')).toBeInTheDocument();
    expect(screen.getByText('(340ms)')).toBeInTheDocument();
    expect(screen.getByText('0.0123')).toBeInTheDocument();
    const failed = screen.getByText('gpt-4o-mini').closest('div.flex.items-center.justify-between') as HTMLElement;
    expect(failed.querySelector('.text-red-500')).not.toBeNull();
    expect(failed).not.toHaveTextContent('ms)');
    expect(failed).not.toHaveTextContent(/^0/);
    expect(screen.getByText('Showing records 1 - 2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    expect(api.calls[0].url).toBe('/rental-keys/rk-1/usage?limit=20&offset=0');
  });

  it('pages forward and back through full pages', async () => {
    const user = userEvent.setup();
    api.on('GET', '/rental-keys/rk-1/usage?limit=20&offset=0', { data: { usage: page(0, 20) } });
    api.on('GET', '/rental-keys/rk-1/usage?limit=20&offset=20', { data: { usage: page(20, 3) } });
    render(<RentalKeyUsageModal isOpen onClose={vi.fn()} rentalKey={rentalKey()} />);

    expect(await screen.findByText('Showing records 1 - 20')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(await screen.findByText('Showing records 21 - 23')).toBeInTheDocument();
    expect(screen.getByText('model-21')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    await user.click(screen.getByRole('button', { name: 'Previous' }));
    expect(await screen.findByText('Showing records 1 - 20')).toBeInTheDocument();
    expect(api.calls.map((c) => c.url)).toEqual([
      '/rental-keys/rk-1/usage?limit=20&offset=0',
      '/rental-keys/rk-1/usage?limit=20&offset=20',
      '/rental-keys/rk-1/usage?limit=20&offset=0',
    ]);
  });

  it('shows the empty state when there is no usage (or a null list)', async () => {
    api.on('GET', /usage/, { data: { usage: null } });
    render(<RentalKeyUsageModal isOpen onClose={vi.fn()} rentalKey={rentalKey()} />);
    expect(await screen.findByText('No usage records found')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument();
  });

  it('shows API errors', async () => {
    api.on('GET', /usage/, { status: 403, data: { error: 'access denied' } });
    render(<RentalKeyUsageModal isOpen onClose={vi.fn()} rentalKey={rentalKey()} />);
    expect(await screen.findByText('access denied')).toBeInTheDocument();
  });

  it('does not fetch while closed and closes via the header button', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    api.on('GET', /usage/, { data: { usage: [] } });
    const { rerender } = render(<RentalKeyUsageModal isOpen={false} onClose={onClose} rentalKey={rentalKey()} />);
    expect(api.calls).toHaveLength(0);
    expect(screen.queryByText('Usage History')).not.toBeInTheDocument();

    rerender(<RentalKeyUsageModal isOpen onClose={onClose} rentalKey={rentalKey()} />);
    await waitFor(() => expect(api.calls).toHaveLength(1));
    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
