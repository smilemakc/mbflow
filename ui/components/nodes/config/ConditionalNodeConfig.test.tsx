import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { ConditionalNodeConfigComponent } from './ConditionalNodeConfig';
import type { ConditionalNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, renderConfig } from '@/test/nodesHarness';
import { useDagStore } from '@/store/dagStore';

describe('ConditionalNodeConfig', () => {
  const t = nodesEnT();
  const c = t.nodeConfig.conditional;

  it('uses the default condition when none is set and emits it', () => {
    const { last } = renderConfig(ConditionalNodeConfigComponent, {} as ConditionalNodeConfig);
    expect(screen.getByPlaceholderText(c.conditionPlaceholder)).toHaveValue('{{input.value}} > 0');
    expect(last()).toEqual({ condition: '{{input.value}} > 0' });
  });

  it('renders examples, notes and operator reference', () => {
    renderConfig(ConditionalNodeConfigComponent, { condition: 'true' });
    expect(screen.getByText('{{input.value}} > 100')).toBeInTheDocument();
    expect(screen.getByText('len({{input.items}}) > 5')).toBeInTheDocument();
    expect(screen.getByText(c.notes.trueBranch)).toBeInTheDocument();
    expect(screen.getByText(c.operators.logicalOps)).toBeInTheDocument();
  });

  it('emits condition edits and keeps other keys', async () => {
    const { user, last } = renderConfig(ConditionalNodeConfigComponent, { condition: 'a', true_branch: 'n2' });
    const input = screen.getByPlaceholderText(c.conditionPlaceholder);
    await user.type(input, ' && b');
    expect(last()).toEqual({ condition: 'a && b', true_branch: 'n2' });
  });

  it('offers variables after typing {{ and inserts the chosen one', async () => {
    useDagStore.setState({ nodes: [], variables: [] } as any);
    const { last } = renderConfig(ConditionalNodeConfigComponent, { condition: 'x' });
    const input = screen.getByPlaceholderText(c.conditionPlaceholder) as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: '{{exec', selectionStart: 6 } });
    const option = await screen.findByText('Execution ID');
    fireEvent.mouseDown(option);
    expect(last().condition).toBe('{{execution_id}}');
  });

  it('lets the user clear the condition without it snapping back to the default', async () => {
    const { user, last } = renderConfig(ConditionalNodeConfigComponent, { condition: 'a > 1' });
    const input = screen.getByPlaceholderText(c.conditionPlaceholder);
    await user.clear(input);
    expect(input).toHaveValue('');
    expect(last()).toEqual({ condition: '' });
    await user.type(input, 'b');
    expect(last()).toEqual({ condition: 'b' });
  });

  it('resyncs when the parent changes the condition', () => {
    const { setConfig } = renderConfig(ConditionalNodeConfigComponent, { condition: 'a' });
    act(() => setConfig({ condition: 'b > 1' }));
    expect(screen.getByPlaceholderText(c.conditionPlaceholder)).toHaveValue('b > 1');
  });
});
