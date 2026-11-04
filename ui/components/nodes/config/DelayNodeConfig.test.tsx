import { describe, expect, it } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { DelayNodeConfig } from './DelayNodeConfig';
import type { DelayNodeConfig as DelayConfig } from '@/types/nodeConfigs';
import { nodesEnT, renderConfig } from '@/test/nodesHarness';

describe('DelayNodeConfig', () => {
  const t = nodesEnT();
  const d = t.nodeConfig.delay;

  it('renders duration, unit, description and a live preview', () => {
    renderConfig(DelayNodeConfig, { duration: 5, unit: 'minutes', description: 'wait' } as DelayConfig);
    expect(screen.getByRole('heading', { name: d.title })).toBeInTheDocument();
    expect(screen.getByRole('spinbutton')).toHaveValue(5);
    expect(screen.getByRole('combobox')).toHaveValue('minutes');
    expect(screen.getByPlaceholderText(d.descriptionPlaceholder)).toHaveValue('wait');
    expect(screen.getByText('5 minutes')).toBeInTheDocument();
  });

  it('falls back to 0 seconds when the config is empty', () => {
    renderConfig(DelayNodeConfig, {} as DelayConfig);
    expect(screen.getByRole('spinbutton')).toHaveValue(0);
    expect(screen.getByRole('combobox')).toHaveValue('seconds');
    expect(screen.getByText('0 seconds')).toBeInTheDocument();
  });

  it('emits duration, clamping negatives and non-numbers to 0', () => {
    const { last } = renderConfig(DelayNodeConfig, { duration: 1, unit: 'seconds' } as DelayConfig);
    const input = screen.getByRole('spinbutton');
    fireEvent.change(input, { target: { value: '42' } });
    expect(last()).toEqual({ duration: 42, unit: 'seconds' });
    fireEvent.change(input, { target: { value: '-3' } });
    expect(last().duration).toBe(0);
    fireEvent.change(input, { target: { value: '' } });
    expect(last().duration).toBe(0);
  });

  it('emits unit and description changes', async () => {
    const { user, last } = renderConfig(DelayNodeConfig, { duration: 2, unit: 'seconds' } as DelayConfig);
    await user.selectOptions(screen.getByRole('combobox'), 'hours');
    expect(last()).toEqual({ duration: 2, unit: 'hours' });
    expect(screen.getByText('2 hours')).toBeInTheDocument();
    await user.type(screen.getByPlaceholderText(d.descriptionPlaceholder), 'Hi');
    expect(last()).toEqual({ duration: 2, unit: 'hours', description: 'Hi' });
  });
});
