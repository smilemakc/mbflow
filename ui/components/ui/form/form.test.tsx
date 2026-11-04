import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { Checkbox, FormField, NumberInput, Select, TextInput, Textarea } from './index';
import { configStyles } from '@/styles/configStyles';

// Stand-in for the builder's autocomplete so the variable mode is observable in isolation.
vi.mock('@/components/builder/VariableAutocomplete', () => ({
  VariableAutocomplete: (props: { type?: string; value: string; onChange: (v: string) => void; rows?: number; className?: string; placeholder?: string }) => (
    <textarea
      data-testid={`autocomplete-${props.type}`}
      data-rows={props.rows}
      className={props.className}
      placeholder={props.placeholder}
      value={props.value}
      onChange={(e) => props.onChange(e.target.value)}
    />
  ),
}));

describe('FormField', () => {
  it('renders label, required marker and hint', () => {
    render(
      <FormField label="API Key" hint="Your key" required htmlFor="k" className="wrap">
        <input id="k" />
      </FormField>,
    );
    expect(screen.getByLabelText(/API Key/)).toBeInTheDocument();
    expect(screen.getByText('*')).toBeInTheDocument();
    expect(screen.getByText('Your key')).toBeInTheDocument();
  });

  it('shows the error instead of the hint and supports custom label classes', () => {
    const { container } = render(
      <FormField label="Name" hint="hint" error="Required" labelClassName="lbl">
        <input />
      </FormField>,
    );
    expect(screen.getByText('Required')).toBeInTheDocument();
    expect(screen.queryByText('hint')).toBeNull();
    expect(container.querySelector('label')!.className).toBe('lbl');
  });

  it('can render without a label', () => {
    const { container } = render(
      <FormField>
        <input />
      </FormField>,
    );
    expect(container.querySelector('label')).toBeNull();
  });
});

describe('TextInput', () => {
  it('renders a native input with passthrough attributes', () => {
    const onChange = vi.fn();
    render(
      <TextInput value="a" onChange={onChange} type="password" id="pw" name="pw" required minLength={2} maxLength={9} pattern="\w+" autoComplete="off" placeholder="Password" />,
    );
    const input = screen.getByPlaceholderText('Password') as HTMLInputElement;
    expect(input).toHaveAttribute('type', 'password');
    expect(input).toHaveAttribute('maxlength', '9');
    expect(input.className).toBe(configStyles.input);
    fireEvent.change(input, { target: { value: 'abc' } });
    expect(onChange).toHaveBeenCalledWith('abc');
  });

  it('switches to variable autocomplete', () => {
    const onChange = vi.fn();
    render(<TextInput value="" onChange={onChange} enableVariables className="custom" />);
    const el = screen.getByTestId('autocomplete-input');
    expect(el.className).toBe('custom');
    fireEvent.change(el, { target: { value: '{{x}}' } });
    expect(onChange).toHaveBeenCalledWith('{{x}}');
  });
});

describe('Textarea', () => {
  it('renders plain and monospace textareas', () => {
    const onChange = vi.fn();
    const { container, rerender } = render(<Textarea value="v" onChange={onChange} />);
    const ta = container.querySelector('textarea')!;
    expect(ta.rows).toBe(4);
    expect(ta.className).toBe(configStyles.textarea);
    fireEvent.change(ta, { target: { value: 'w' } });
    expect(onChange).toHaveBeenCalledWith('w');

    rerender(<Textarea value="v" onChange={onChange} monospace rows={8} disabled />);
    expect(container.querySelector('textarea')!.className).toBe(configStyles.textareaMonospace);
    expect(container.querySelector('textarea')).toBeDisabled();
  });

  it('switches to variable autocomplete', () => {
    render(<Textarea value="" onChange={() => {}} enableVariables rows={6} />);
    expect(screen.getByTestId('autocomplete-textarea')).toHaveAttribute('data-rows', '6');
  });
});

describe('Select', () => {
  it('normalises string and object options and shows a placeholder', () => {
    const onChange = vi.fn();
    const { rerender } = render(<Select value="" onChange={onChange} options={['GET', 'POST']} placeholder="Method" />);
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Method', 'GET', 'POST']);
    expect(screen.getByRole('option', { name: 'Method' })).toBeDisabled();
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'POST' } });
    expect(onChange).toHaveBeenCalledWith('POST');

    rerender(<Select value="a" onChange={onChange} options={[{ value: 'a', label: 'Alpha' }]} className="sel" disabled />);
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Alpha']);
    expect(screen.getByRole('combobox')).toBeDisabled();
    expect(screen.getByRole('combobox').className).toBe('sel');
  });
});

describe('Checkbox', () => {
  it('works with and without a label', () => {
    const onChange = vi.fn();
    const { rerender } = render(<Checkbox checked={false} onChange={onChange} label="Enable" />);
    fireEvent.click(screen.getByLabelText('Enable'));
    expect(onChange).toHaveBeenCalledWith(true);

    rerender(<Checkbox checked onChange={onChange} className="cb" disabled />);
    const box = screen.getByRole('checkbox');
    expect(box).toBeChecked();
    expect(box).toBeDisabled();
    expect(box.className).toContain('cb');
    rerender(<Checkbox checked onChange={onChange} />);
    fireEvent.click(screen.getByRole('checkbox'));
    expect(onChange).toHaveBeenLastCalledWith(false);
  });
});

describe('NumberInput', () => {
  it('parses numbers, maps empty input to min and ignores garbage', () => {
    const onChange = vi.fn();
    const { rerender } = render(<NumberInput value={1.5} onChange={onChange} min={1} max={5} step={0.5} placeholder="n" />);
    const input = screen.getByPlaceholderText('n') as HTMLInputElement;
    expect(input.value).toBe('1.5');
    fireEvent.change(input, { target: { value: '2.5' } });
    expect(onChange).toHaveBeenLastCalledWith(2.5);
    fireEvent.change(input, { target: { value: '' } });
    expect(onChange).toHaveBeenLastCalledWith(1);

    rerender(<NumberInput value={undefined} onChange={onChange} placeholder="n" className="num" />);
    expect(input.value).toBe('');
    expect(input.className).toBe('num');
    rerender(<NumberInput value={4} onChange={onChange} placeholder="n" />);
    fireEvent.change(input, { target: { value: '' } });
    expect(onChange).toHaveBeenLastCalledWith(0);
  });

  it('ignores values that do not parse', () => {
    const onChange = vi.fn();
    render(<NumberInput value={1} onChange={onChange} />);
    const input = screen.getByRole('spinbutton');
    // jsdom sanitises invalid text for number inputs to '', so force the raw value.
    Object.defineProperty(input, 'value', { configurable: true, get: () => 'abc', set: () => {} });
    fireEvent.change(input);
    fireEvent.input(input);
    expect(onChange).not.toHaveBeenCalledWith(NaN);
  });
});
