import { describe, expect, it } from 'vitest';
import { act, screen } from '@testing-library/react';
import { MergeNodeConfigComponent } from './MergeNodeConfig';
import type { MergeNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, renderConfig } from '@/test/nodesHarness';

describe('MergeNodeConfig', () => {
  const t = nodesEnT();
  const m = t.nodeConfig.merge;

  it('shows the strategy and hides the custom expression unless strategy is custom', async () => {
    const { user, last, onChange } = renderConfig(MergeNodeConfigComponent, { merge_strategy: 'all' });
    expect(onChange).not.toHaveBeenCalled();
    const select = screen.getByRole('combobox');
    expect(select).toHaveValue('all');
    expect(screen.queryByPlaceholderText(m.customExpressionPlaceholder)).toBeNull();

    await user.selectOptions(select, 'first');
    expect(last()).toEqual({ merge_strategy: 'first' });

    await user.selectOptions(select, 'custom');
    expect(last()).toEqual({ merge_strategy: 'custom' });
    const expr = screen.getByPlaceholderText(m.customExpressionPlaceholder);
    await user.type(expr, 'a + b');
    expect(last()).toEqual({ merge_strategy: 'custom', custom_expression: 'a + b' });
    expect(screen.getByText(m.customExpressionHint)).toBeInTheDocument();
  });

  it('defaults to "all" for an empty config and emits it once (no update loop)', () => {
    const { onChange } = renderConfig(MergeNodeConfigComponent, {} as MergeNodeConfig);
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenLastCalledWith({ merge_strategy: 'all' });
    expect(screen.getByRole('combobox')).toHaveValue('all');
    expect(screen.getByText(/Merge Strategy Examples/)).toBeInTheDocument();
  });

  it('resyncs from the parent', () => {
    const { setConfig } = renderConfig(MergeNodeConfigComponent, { merge_strategy: 'all' });
    act(() => setConfig({ merge_strategy: 'custom', custom_expression: 'x' }));
    expect(screen.getByRole('combobox')).toHaveValue('custom');
    expect(screen.getByPlaceholderText(m.customExpressionPlaceholder)).toHaveValue('x');
  });
});
