import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';
import { MethodBadge } from './MethodBadge';
import { KeyValueRows } from './KeyValueRows';
import { BaseUrlField } from './BaseUrlField';
import { useUIStore } from '@/store/uiStore';

beforeEach(() => {
  useUIStore.setState({ language: 'en' });
});

describe('MethodBadge', () => {
  it('upper-cases the method and colours known methods', () => {
    render(
      <>
        <MethodBadge method="get" className="extra" />
        <MethodBadge method="trace" />
      </>,
    );
    expect(screen.getByText('GET').className).toContain('emerald');
    expect(screen.getByText('GET').className).toContain('extra');
    expect(screen.getByText('TRACE').className).toContain('slate');
  });
});

describe('KeyValueRows', () => {
  const Harness: React.FC<{ initial: Record<string, string>; onChange: (v: Record<string, string>) => void; resetKey?: string; enableVariables?: boolean }> = ({
    initial,
    onChange,
    resetKey,
    enableVariables,
  }) => {
    const [value, setValue] = useState(initial);
    return (
      <KeyValueRows
        value={value}
        onChange={(v) => {
          setValue(v);
          onChange(v);
        }}
        addLabel="Add header"
        keyPlaceholder="Header"
        valuePlaceholder="Value"
        resetKey={resetKey}
        enableVariables={enableVariables}
      />
    );
  };

  it('edits, adds and removes rows, ignoring rows without a key', async () => {
    const onChange = vi.fn();
    render(<Harness initial={{ Accept: 'json' }} onChange={onChange} />);
    expect(screen.getByDisplayValue('Accept')).toBeInTheDocument();
    expect(screen.getByDisplayValue('json')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: /Add header/ }));
    const keys = screen.getAllByPlaceholderText('Header');
    const values = screen.getAllByPlaceholderText('Value');
    expect(keys).toHaveLength(2);

    // A row with only a value stays visible but is not part of the value.
    await userEvent.type(values[1], 'v');
    expect(onChange).toHaveBeenLastCalledWith({ Accept: 'json' });
    expect(screen.getAllByPlaceholderText('Header')).toHaveLength(2);

    await userEvent.type(keys[1], ' X-Id ');
    expect(onChange).toHaveBeenLastCalledWith({ Accept: 'json', 'X-Id': 'v' });

    await userEvent.click(screen.getAllByRole('button', { name: 'Remove' })[0]);
    expect(onChange).toHaveBeenLastCalledWith({ 'X-Id': 'v' });
    expect(screen.queryByDisplayValue('Accept')).not.toBeInTheDocument();
  });

  it('rebuilds rows only when the reset key changes', () => {
    const { rerender } = render(
      <KeyValueRows value={{ a: '1' }} onChange={vi.fn()} addLabel="Add" resetKey="one" />,
    );
    rerender(<KeyValueRows value={{ b: '2' }} onChange={vi.fn()} addLabel="Add" resetKey="one" />);
    expect(screen.getByDisplayValue('a')).toBeInTheDocument();
    rerender(<KeyValueRows value={{ b: '2' }} onChange={vi.fn()} addLabel="Add" resetKey="two" />);
    expect(screen.getByDisplayValue('b')).toBeInTheDocument();
    expect(screen.queryByDisplayValue('a')).not.toBeInTheDocument();
  });

  it('uses the variable autocomplete for values when enabled', () => {
    const onChange = vi.fn();
    render(<Harness initial={{ Auth: 'x' }} onChange={onChange} enableVariables />);
    fireEvent.change(screen.getByDisplayValue('x'), { target: { value: '{{env.TOKEN}}' } });
    expect(onChange).toHaveBeenLastCalledWith({ Auth: '{{env.TOKEN}}' });
  });
});

describe('BaseUrlField', () => {
  const servers = [
    { url: 'https://api.example.com/v1', description: 'Production' },
    { url: 'http://localhost:8080' },
    { url: '/relative' },
  ];

  it('offers absolute servers as choices and warns when empty', async () => {
    const onChange = vi.fn();
    const { rerender } = render(<BaseUrlField value="" onChange={onChange} servers={servers} />);
    expect(screen.getByText('Base URL')).toBeInTheDocument();
    expect(screen.getByText(/Base URL is not set/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '/relative' })).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'https://api.example.com/v1' }));
    expect(onChange).toHaveBeenCalledWith('https://api.example.com/v1');
    expect(screen.getByRole('button', { name: 'https://api.example.com/v1' })).toHaveAttribute('title', 'Production');

    rerender(<BaseUrlField value="http://localhost:8080" onChange={onChange} servers={servers} />);
    expect(screen.queryByText(/Base URL is not set/)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'http://localhost:8080' }).className).toContain('border-blue-500');

    fireEvent.change(screen.getByPlaceholderText('https://api.example.com/v1'), { target: { value: 'https://x.io' } });
    expect(onChange).toHaveBeenLastCalledWith('https://x.io');
  });

  it('shows no choices without absolute servers', () => {
    render(<BaseUrlField value="https://a" onChange={vi.fn()} servers={[{ url: '/v2' }]} />);
    expect(screen.queryAllByRole('button')).toHaveLength(0);
  });
});
