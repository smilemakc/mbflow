import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { TransformNodeConfigComponent } from './TransformNodeConfig';
import type { TransformNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('TransformNodeConfig', () => {
  const t = nodesEnT();
  const tr = t.nodeConfig.transform;

  it('defaults to passthrough with no extra fields', () => {
    render(<TransformNodeConfigComponent config={undefined as unknown as TransformNodeConfig} onChange={() => {}} />);
    expect(screen.getByRole('combobox')).toHaveValue('passthrough');
    expect(screen.getByText(tr.passthroughDesc, { exact: false })).toBeInTheDocument();
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(nodesField(tr.timeout)).toHaveValue(10);
  });

  it('switches type and shows the matching editor', async () => {
    const { user, last } = renderConfig(TransformNodeConfigComponent, { type: 'passthrough' } as TransformNodeConfig);
    const select = screen.getByRole('combobox');
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Passthrough', 'Template', 'Expression', 'Jq']);

    await user.selectOptions(select, 'template');
    expect(last()).toEqual({ type: 'template', template: '', expression: '', filter: '.', timeout_seconds: 10 });
    expect(screen.getByText(tr.templateDesc, { exact: false })).toBeInTheDocument();
    await user.type(screen.getByPlaceholderText(tr.templatePlaceholder), 'Hi');
    expect(last().template).toBe('Hi');

    await user.selectOptions(select, 'expression');
    expect(screen.getByText(tr.expressionDesc, { exact: false })).toBeInTheDocument();
    await user.type(screen.getByPlaceholderText(tr.expressionPlaceholder), 'x*2');
    expect(last()).toMatchObject({ type: 'expression', expression: 'x*2', template: 'Hi' });

    await user.selectOptions(select, 'jq');
    expect(screen.getByText(tr.jqDesc, { exact: false })).toBeInTheDocument();
    const jq = screen.getByPlaceholderText(tr.jqPlaceholder);
    expect(jq).toHaveValue('.');
    await user.type(jq, 'items');
    expect(last().filter).toBe('.items');

    fireEvent.change(nodesField(tr.timeout), { target: { value: '30' } });
    expect(last().timeout_seconds).toBe(30);
  });
});
