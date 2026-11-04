import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ResourceList } from './ResourceList';
import { ResourceCard } from './ResourceCard';
import { useUIStore } from '@/store/uiStore';
import { storage } from '@/test/resourcesFixtures';

const cardFor = (name: string) => screen.getByRole('heading', { name }).closest('div.rounded-xl') as HTMLElement;

describe('ResourceList', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('shows the empty state with two create buttons', async () => {
    const onCreate = vi.fn();
    render(<ResourceList resources={[]} onCreateClick={onCreate} onDeleteResource={vi.fn()} onViewFiles={vi.fn()} />);
    expect(screen.getByText('No storage resources yet')).toBeInTheDocument();
    const buttons = screen.getAllByRole('button', { name: 'Create Storage' });
    expect(buttons).toHaveLength(2);
    await userEvent.click(buttons[1]);
    await userEvent.click(buttons[0]);
    expect(onCreate).toHaveBeenCalledTimes(2);
  });

  it('renders a card per resource and wires view/delete callbacks', async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    const onView = vi.fn();
    const a = storage({ id: 'a', name: 'Alpha' });
    const b = storage({ id: 'b', name: 'Beta', description: '' });
    render(<ResourceList resources={[a, b]} onCreateClick={vi.fn()} onDeleteResource={onDelete} onViewFiles={onView} />);

    expect(screen.queryByText('No storage resources yet')).not.toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'View Files' })).toHaveLength(2);

    await user.click(within(cardFor('Beta')).getByRole('button', { name: 'View Files' }));
    expect(onView).toHaveBeenCalledWith(b);
    await user.click(screen.getByRole('button', { name: 'Delete Alpha' }));
    expect(onDelete).toHaveBeenCalledWith('a', 'Alpha');
  });
});

describe('ResourceCard', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  const renderCard = (overrides: Parameters<typeof storage>[0]) =>
    render(<ResourceCard resource={storage(overrides)} onDelete={vi.fn()} onViewFiles={vi.fn()} />);

  it('shows name, description, usage and created date for a healthy resource', () => {
    const { container } = renderCard({});
    expect(screen.getByText('Main bucket')).toBeInTheDocument();
    expect(screen.getByText('active').className).toContain('green');
    expect(screen.getByText('512 MB / 1 GB')).toBeInTheDocument();
    expect(screen.getByText('3 files')).toBeInTheDocument();
    const pct = screen.getByText('50.0% used');
    expect(pct.className).toContain('text-slate-600');
    const bar = container.querySelector('.h-2.rounded-full.transition-all') as HTMLElement;
    expect(bar.style.width).toBe('50%');
    expect(bar.className).toContain('bg-blue-600');
    expect(screen.getByText(/Created Mar 15, 2026/)).toBeInTheDocument();
  });

  it('warns when usage is above 75%', () => {
    const { container } = renderCard({ usage_percent: 80, file_count: 1, description: '' });
    expect(screen.getByText('1 file')).toBeInTheDocument();
    expect(screen.getByText('80.0% used').className).toContain('yellow');
    expect(container.querySelector('.bg-yellow-500')).not.toBeNull();
    expect(screen.queryByText('Main bucket')).not.toBeInTheDocument();
  });

  it('turns red above 90% and caps the bar at 100%', () => {
    const { container } = renderCard({ usage_percent: 120, status: 'suspended', used_storage_bytes: 0 });
    expect(screen.getByText('120.0% used').className).toContain('red');
    const bar = container.querySelector('.bg-red-600') as HTMLElement;
    expect(bar.style.width).toBe('100%');
    expect(screen.getByText('suspended').className).toContain('gray');
    expect(screen.getByText('0 B / 1 GB')).toBeInTheDocument();
  });
});
