import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TypeSelector } from './TypeSelector';
import { CustomFieldsForm } from './forms';
import { useUIStore } from '@/store/uiStore';

describe('TypeSelector', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('offers every credential type the backend can create, with a description and icon', async () => {
    const onSelect = vi.fn();
    render(<TypeSelector onSelect={onSelect} />);
    const buttons = screen.getAllByRole('button');
    expect(buttons.map((b) => b.querySelector('.font-medium')!.textContent)).toEqual([
      'API Key',
      'Basic Auth',
      'OAuth2',
      'Service Account',
      'Custom',
    ]);
    buttons.forEach((b) => expect(b.querySelector('svg')).not.toBeNull());
    expect(screen.getByText('Simple API key or token for accessing external services')).toBeInTheDocument();

    const user = userEvent.setup();
    for (const b of buttons) await user.click(b);
    expect(onSelect.mock.calls.map((c) => c[0])).toEqual(['api_key', 'basic_auth', 'oauth2', 'service_account', 'custom']);
  });
});

describe('CustomFieldsForm', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('updates rows immutably, adds and removes rows', async () => {
    const user = userEvent.setup();
    const fields = [
      { key: 'a', value: '1' },
      { key: 'b', value: '2' },
    ];
    const snapshot = JSON.parse(JSON.stringify(fields));
    const onFieldsChange = vi.fn();
    render(<CustomFieldsForm fields={fields} onFieldsChange={onFieldsChange} />);

    await user.type(screen.getAllByPlaceholderText('Key')[1], 'x');
    expect(onFieldsChange).toHaveBeenLastCalledWith([{ key: 'a', value: '1' }, { key: 'bx', value: '2' }]);
    await user.type(screen.getAllByPlaceholderText('Value (secret)')[0], '9');
    expect(onFieldsChange).toHaveBeenLastCalledWith([{ key: 'a', value: '19' }, { key: 'b', value: '2' }]);
    expect(fields).toEqual(snapshot); // props are never mutated

    await user.click(screen.getByRole('button', { name: 'Remove field 1' }));
    expect(onFieldsChange).toHaveBeenLastCalledWith([{ key: 'b', value: '2' }]);
    await user.click(screen.getByRole('button', { name: '+ Add Field' }));
    expect(onFieldsChange).toHaveBeenLastCalledWith([...snapshot, { key: '', value: '' }]);
  });
});
