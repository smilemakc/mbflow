import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { FunctionCallNodeConfigComponent } from './FunctionCallNodeConfig';
import type { FunctionCallNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('FunctionCallNodeConfig', () => {
  const t = nodesEnT();
  const f = t.nodeConfig.functionCall;

  it('applies defaults to an empty config', () => {
    const { last } = renderConfig(FunctionCallNodeConfigComponent, {} as FunctionCallNodeConfig);
    expect(last()).toEqual({ function_name: '', arguments: {}, timeout_seconds: 30 });
    expect(nodesField(f.timeout)).toHaveValue(30);
    expect(nodesField(f.arguments, 0)).toHaveValue('{}');
    expect(screen.getByText(f.argumentsHint)).toBeInTheDocument();
  });

  it('edits name and timeout', async () => {
    const { user, last } = renderConfig(FunctionCallNodeConfigComponent, {
      function_name: 'fn',
      arguments: { a: 1 },
      timeout_seconds: 30,
    });
    await user.type(screen.getByPlaceholderText(f.functionNamePlaceholder), '2');
    expect(last()).toEqual({ function_name: 'fn2', arguments: { a: 1 }, timeout_seconds: 30 });
    fireEvent.change(nodesField(f.timeout), { target: { value: '90' } });
    expect(last().timeout_seconds).toBe(90);
  });

  it('parses valid JSON arguments and keeps invalid JSON as raw text', () => {
    const { last } = renderConfig(FunctionCallNodeConfigComponent, {
      function_name: 'fn',
      arguments: {},
      timeout_seconds: 30,
    });
    const args = screen.getByPlaceholderText(f.argumentsPlaceholder);
    fireEvent.change(args, { target: { value: '{"city": "{{input.city}}", "n": 2}' } });
    expect(last().arguments).toEqual({ city: '{{input.city}}', n: 2 });

    fireEvent.change(args, { target: { value: '{"broken": ' } });
    expect(last().arguments).toBe('{"broken": ');
    expect(args).toHaveValue('{"broken": ');
  });

  it('shows string arguments verbatim and resyncs from the parent without emitting the stale config', () => {
    const { setConfig, onChange } = renderConfig(FunctionCallNodeConfigComponent, {
      function_name: 'fn',
      arguments: 'raw text' as any,
      timeout_seconds: 30,
    });
    expect(screen.getByPlaceholderText(f.argumentsPlaceholder)).toHaveValue('raw text');
    onChange.mockClear();
    act(() => setConfig({ function_name: 'other', arguments: { x: true }, timeout_seconds: 5 }));
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByPlaceholderText(f.functionNamePlaceholder)).toHaveValue('other');
    expect(screen.getByPlaceholderText(f.argumentsPlaceholder)).toHaveValue(JSON.stringify({ x: true }, null, 2));
    expect(nodesField(f.timeout)).toHaveValue(5);
  });
});
