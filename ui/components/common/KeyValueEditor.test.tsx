import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { KeyValueEditor } from './KeyValueEditor';

// The real autocomplete pulls in the builder store; a plain input keeps this unit-scoped.
vi.mock('@/components/builder/VariableAutocomplete', () => ({
  VariableAutocomplete: ({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder?: string }) => (
    <input placeholder={placeholder} value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}));

function Harness({ initial, spy }: { initial: Record<string, string>; spy: (v: Record<string, string>) => void }) {
  const [value, setValue] = useState(initial);
  return (
    <KeyValueEditor
      value={value}
      onChange={(v) => {
        spy(v);
        setValue(v);
      }}
      placeholderKey="Header"
      placeholderValue="Header value"
      itemLabel="Header"
    />
  );
}

describe('KeyValueEditor', () => {
  it('starts with one empty row and builds an object as the user types', () => {
    const spy = vi.fn();
    render(<Harness initial={{}} spy={spy} />);
    expect(screen.getAllByPlaceholderText('Header')).toHaveLength(1);

    fireEvent.change(screen.getByPlaceholderText('Header value'), { target: { value: 'v-first' } });
    // Rows without a key are not emitted.
    expect(spy).toHaveBeenLastCalledWith({});

    fireEvent.change(screen.getByPlaceholderText('Header'), { target: { value: 'X-Token' } });
    expect(spy).toHaveBeenLastCalledWith({ 'X-Token': 'v-first' });
  });

  it('adds, edits and removes rows', () => {
    const spy = vi.fn();
    render(<Harness initial={{ a: '1', b: '2' }} spy={spy} />);
    expect(screen.getAllByPlaceholderText('Header').map((i) => (i as HTMLInputElement).value)).toEqual(['a', 'b']);

    fireEvent.click(screen.getByRole('button', { name: /add header/i }));
    expect(screen.getAllByPlaceholderText('Header')).toHaveLength(3);
    expect(spy).not.toHaveBeenCalled();

    fireEvent.change(screen.getAllByPlaceholderText('Header value')[1], { target: { value: '22' } });
    expect(spy).toHaveBeenLastCalledWith({ a: '1', b: '22' });

    fireEvent.click(screen.getAllByTitle('Remove')[0]);
    expect(spy).toHaveBeenLastCalledWith({ b: '22' });
    // The blank row added earlier is kept.
    expect(screen.getAllByPlaceholderText('Header').map((i) => (i as HTMLInputElement).value)).toEqual(['b', '']);

    fireEvent.click(screen.getAllByTitle('Remove')[1]);
    fireEvent.click(screen.getByTitle('Remove'));
    expect(spy).toHaveBeenLastCalledWith({});
    expect(screen.getAllByPlaceholderText('Header')).toHaveLength(1);
  });

  it('keeps a new row while its value is typed before its key', () => {
    const spy = vi.fn();
    render(<Harness initial={{ a: '1' }} spy={spy} />);
    fireEvent.click(screen.getByRole('button', { name: /add header/i }));
    fireEvent.change(screen.getAllByPlaceholderText('Header value')[1], { target: { value: 'later' } });
    expect(spy).toHaveBeenLastCalledWith({ a: '1' });
    expect(screen.getAllByPlaceholderText('Header value').map((i) => (i as HTMLInputElement).value)).toEqual(['1', 'later']);
    fireEvent.change(screen.getAllByPlaceholderText('Header')[1], { target: { value: 'b' } });
    expect(spy).toHaveBeenLastCalledWith({ a: '1', b: 'later' });
  });

  it('resyncs rows when the value changes from outside', () => {
    const { rerender } = render(<KeyValueEditor value={{ a: '1' }} onChange={() => {}} />);
    rerender(<KeyValueEditor value={{ z: '9' }} onChange={() => {}} />);
    expect((screen.getByPlaceholderText('Key') as HTMLInputElement).value).toBe('z');
  });

  it('uses default labels', () => {
    render(<KeyValueEditor value={{}} onChange={() => {}} />);
    expect(screen.getByPlaceholderText('Key')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Value')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add item/i })).toBeInTheDocument();
  });
});
