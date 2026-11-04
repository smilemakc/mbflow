import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { SubWorkflowNodeConfigComponent } from './SubWorkflowNodeConfig';
import type { SubWorkflowNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('SubWorkflowNodeConfig', () => {
  const t = nodesEnT();
  const s = (t.nodeConfig as any).subWorkflow;

  it('fills defaults for an empty config and emits them', () => {
    const { onChange } = renderConfig(SubWorkflowNodeConfigComponent, {} as SubWorkflowNodeConfig);
    expect(onChange).toHaveBeenLastCalledWith({
      workflow_id: '',
      for_each: 'input.items',
      item_var: 'item',
      max_parallelism: 0,
      on_error: 'fail_fast',
    });
    expect(screen.getByText(s.howItWorks)).toBeInTheDocument();
    expect(screen.getAllByRole('listitem')).toHaveLength(s.howItWorksItems.length);
  });

  it('emits edits of every field', async () => {
    const initial: SubWorkflowNodeConfig = {
      workflow_id: 'wf',
      for_each: 'input.items',
      item_var: 'item',
      max_parallelism: 0,
      on_error: 'fail_fast',
    };
    const { user, last, onChange } = renderConfig(SubWorkflowNodeConfigComponent, initial);
    expect(onChange).not.toHaveBeenCalled();

    await user.type(nodesField(s.workflowId), '-1');
    expect(last().workflow_id).toBe('wf-1');

    await user.clear(nodesField(s.forEach));
    await user.type(nodesField(s.forEach), 'input.rows');
    expect(last().for_each).toBe('input.rows');

    await user.clear(nodesField(s.itemVar));
    await user.type(nodesField(s.itemVar), 'row');
    expect(last().item_var).toBe('row');

    fireEvent.change(nodesField(s.maxParallelism), { target: { value: '4' } });
    expect(last().max_parallelism).toBe(4);
    fireEvent.change(nodesField(s.maxParallelism), { target: { value: '' } });
    expect(last().max_parallelism).toBe(0);

    await user.selectOptions(nodesField<HTMLSelectElement>(s.onError), 'collect_partial');
    expect(last()).toEqual({
      workflow_id: 'wf-1',
      for_each: 'input.rows',
      item_var: 'row',
      max_parallelism: 0,
      on_error: 'collect_partial',
    });
  });

  it('resyncs from the parent', () => {
    const { setConfig } = renderConfig(SubWorkflowNodeConfigComponent, { workflow_id: 'a', for_each: 'x' } as SubWorkflowNodeConfig);
    act(() => setConfig({ workflow_id: 'b', for_each: 'y', item_var: 'it', max_parallelism: 3, on_error: 'collect_partial' }));
    expect(nodesField(s.workflowId)).toHaveValue('b');
    expect(nodesField(s.maxParallelism)).toHaveValue(3);
    expect(nodesField<HTMLSelectElement>(s.onError)).toHaveValue('collect_partial');
  });
});
