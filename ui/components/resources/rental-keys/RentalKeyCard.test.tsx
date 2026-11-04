import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RentalKeyCard } from './RentalKeyCard';
import { useUIStore } from '@/store/uiStore';
import { rentalKey } from '@/test/resourcesFixtures';
import type { RentalKey } from '@/services/rentalKeyService';

const limitRow = (label: string) => screen.getByText(label).closest('div')!.parentElement as HTMLElement;

describe('RentalKeyCard', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  const renderCard = (overrides: Partial<RentalKey> = {}) => {
    const onViewUsage = vi.fn();
    const key = rentalKey(overrides);
    render(<RentalKeyCard rentalKey={key} onViewUsage={onViewUsage} />);
    return { key, onViewUsage };
  };

  it('shows provider, status, aggregate stats and dates without any limits', () => {
    renderCard({ last_used_at: '2026-04-02T10:00:00Z' });
    expect(screen.getByRole('heading', { name: 'Team GPT key' })).toBeInTheDocument();
    expect(screen.getByText('OpenAI')).toBeInTheDocument();
    expect(screen.getByText('Shared key')).toBeInTheDocument();
    expect(screen.getByText('active').className).toContain('green');
    expect(screen.getByText('2.50K')).toBeInTheDocument(); // total_requests
    expect(screen.getByText('4.00M')).toBeInTheDocument(); // total tokens
    expect(screen.getByText('$12.35')).toBeInTheDocument();
    expect(screen.getByText(/Created Mar 15, 2026/)).toBeInTheDocument();
    expect(screen.getByText(/Apr 2, 2026/)).toBeInTheDocument();
    expect(screen.queryByText('Daily Requests')).not.toBeInTheDocument();
    expect(screen.queryByText('Monthly Tokens')).not.toBeInTheDocument();
  });

  it('does not render a stray 0 for zero limits', () => {
    const { container } = render(
      <RentalKeyCard rentalKey={rentalKey({ daily_request_limit: 0, monthly_token_limit: 0 })} onViewUsage={vi.fn()} />,
    );
    const limits = container.querySelector('.space-y-3.mb-3')!;
    expect(limits.textContent).toBe('');
  });

  it('renders limit bars with colour thresholds', () => {
    renderCard({
      daily_request_limit: 100,
      requests_today: 50,
      monthly_token_limit: 10_000,
      tokens_this_month: 8_000,
      provider: 'anthropic',
      status: 'suspended',
      description: undefined,
    });
    expect(screen.getByText('Anthropic')).toBeInTheDocument();
    expect(screen.getByText('suspended').className).toContain('gray');

    const daily = limitRow('Daily Requests');
    expect(daily).toHaveTextContent('50 / 100');
    expect(daily).toHaveTextContent('50.0%');
    expect(daily.querySelector('.bg-blue-600')).not.toBeNull();
    expect(daily.querySelector('.text-yellow-500')).toBeNull(); // not near limit

    const monthly = limitRow('Monthly Tokens');
    expect(monthly).toHaveTextContent('8.00K / 10.00K');
    expect(monthly.querySelector('.bg-yellow-500')).not.toBeNull();
    expect(monthly.querySelector('.text-yellow-600')).not.toBeNull();
    expect(monthly.querySelector('.text-yellow-500')).not.toBeNull(); // near-limit warning icon
  });

  it('turns red above 90% and caps progress at 100%', () => {
    renderCard({ daily_request_limit: 10, requests_today: 25, provider: 'google_ai' });
    expect(screen.getByText('Google AI')).toBeInTheDocument();
    const daily = limitRow('Daily Requests');
    expect(daily).toHaveTextContent('25 / 10');
    expect(daily).toHaveTextContent('100.0%');
    const bar = daily.querySelector('.bg-red-600') as HTMLElement;
    expect(bar.style.width).toBe('100%');
    expect(daily.querySelector('.text-red-600')).not.toBeNull();
  });

  it('falls back to a neutral colour for unknown providers and calls onViewUsage', async () => {
    const { key, onViewUsage } = renderCard({ provider: 'mistral' as never });
    expect(screen.getByText('mistral')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'View Usage' }));
    expect(onViewUsage).toHaveBeenCalledWith(key);
  });
});
