import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { VariableAutocomplete } from './VariableAutocomplete';
import { useDagStore } from '@/store/dagStore';
import { VariableSource, VariableType } from '@/types';
import { makeNode, resetBuilderStores } from '@/test/builderFlowMocks';

const Harness = ({ type, initial = '', onChange }: { type?: 'input' | 'textarea'; initial?: string; onChange?: (v: string) => void }) => {
  const [value, setValue] = useState(initial);
  return (
    <VariableAutocomplete
      type={type}
      value={value}
      placeholder="Type here"
      onChange={(v) => {
        setValue(v);
        onChange?.(v);
      }}
    />
  );
};

beforeEach(() => {
  resetBuilderStores();
  useDagStore.setState({
    nodes: [makeNode('http')],
    variables: [
      { id: 'v1', key: 'str', name: 'Str', type: VariableType.STRING, source: VariableSource.INPUT },
      { id: 'v2', key: 'num', name: 'Num', type: VariableType.NUMBER, source: VariableSource.INPUT },
      { id: 'v3', key: 'bool', name: 'Bool', type: VariableType.BOOLEAN, source: VariableSource.INPUT },
      { id: 'v4', key: 'obj', name: 'Obj', type: VariableType.OBJECT, source: VariableSource.INPUT },
      { id: 'v5', key: 'arr', name: 'Arr', type: 'array' as VariableType, source: VariableSource.INPUT },
    ] as never,
  });
});

const suggestions = () => screen.queryAllByRole('button').map((b) => b.textContent);

describe('VariableAutocomplete', () => {
  it('opens suggestions after {{, filters them and inserts the chosen variable', async () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    const input = screen.getByPlaceholderText('Type here');
    await userEvent.type(input, 'Hi {{{{');
    expect(screen.getByText('Suggested Variables')).toBeInTheDocument();
    expect(suggestions()).toHaveLength(8);
    expect(suggestions()).toContain('Execution ID{{execution_id}}');
    expect(suggestions()).toContain('Label http Output{{step_http.output}}');

    await userEvent.type(input, 'nu');
    expect(suggestions()).toEqual(['Num{{num}}']);

    fireEvent.mouseDown(screen.getByRole('button', { name: /Num/ }));
    expect(onChange).toHaveBeenLastCalledWith('Hi {{num}}');
    expect(input).toHaveValue('Hi {{num}}');
    expect(screen.queryByText('Suggested Variables')).not.toBeInTheDocument();
    await waitFor(() => expect(input).toHaveFocus());
    expect((input as HTMLInputElement).selectionStart).toBe(10);
  });

  it('closes when the trigger is gone, on Escape and on blur', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      render(<Harness type="textarea" />);
      const area = screen.getByPlaceholderText('Type here');
      expect(area.tagName).toBe('TEXTAREA');
      fireEvent.change(area, { target: { value: '{{', selectionStart: 2 } });
      expect(screen.getByText('Suggested Variables')).toBeInTheDocument();
      fireEvent.keyDown(area, { key: 'a' });
      expect(screen.getByText('Suggested Variables')).toBeInTheDocument();
      fireEvent.keyDown(area, { key: 'Escape' });
      expect(screen.queryByText('Suggested Variables')).not.toBeInTheDocument();
      fireEvent.keyDown(area, { key: 'Escape' });

      fireEvent.change(area, { target: { value: '{{x y', selectionStart: 5 } });
      expect(screen.queryByText('Suggested Variables')).not.toBeInTheDocument();

      fireEvent.change(area, { target: { value: '{{', selectionStart: 2 } });
      fireEvent.blur(area);
      expect(screen.getByText('Suggested Variables')).toBeInTheDocument();
      act(() => vi.advanceTimersByTime(250));
      expect(screen.queryByText('Suggested Variables')).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('hides the dropdown when nothing matches', () => {
    render(<Harness />);
    const input = screen.getByPlaceholderText('Type here');
    fireEvent.change(input, { target: { value: '{{zzz', selectionStart: 5 } });
    expect(screen.queryByText('Suggested Variables')).not.toBeInTheDocument();
  });

  it('closes suggestions when the cursor is before the trigger and tolerates a missing cursor position', () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    const input = screen.getByPlaceholderText('Type here');
    fireEvent.change(input, { target: { value: '{{', selectionStart: 2 } });
    const option = screen.getByRole('button', { name: /Timestamp/ });
    fireEvent.change(input, { target: { value: 'ab{{', selectionStart: 0 } });
    expect(screen.queryByText('Suggested Variables')).not.toBeInTheDocument();
    expect(option).not.toBeInTheDocument();
    onChange.mockClear();
    fireEvent.change(input, { target: { value: 'x{{', selectionStart: undefined } });
    expect(onChange).toHaveBeenCalledWith('x{{');
  });
});
