import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { CSVToJSONNodeConfigComponent } from './CSVToJSONNodeConfig';
import type { CSVToJSONNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('CSVToJSONNodeConfig', () => {
  const t = nodesEnT();
  const c = t.nodeConfig.csvToJson;

  it('renders defaults for a missing config', () => {
    render(<CSVToJSONNodeConfigComponent config={undefined as unknown as CSVToJSONNodeConfig} onChange={() => {}} />);
    expect(nodesField<HTMLSelectElement>(c.delimiter)).toHaveValue(',');
    expect(nodesField(c.hasHeader)).toBeChecked();
    expect(nodesField(c.trimSpaces)).toBeChecked();
    expect(nodesField(c.skipEmptyRows)).toBeChecked();
    expect(screen.queryByText(c.customHeaders)).toBeNull();
  });

  it('emits delimiter, input key and processing options', async () => {
    const { user, last } = renderConfig(CSVToJSONNodeConfigComponent, {} as CSVToJSONNodeConfig);
    await user.selectOptions(nodesField<HTMLSelectElement>(c.delimiter), ';');
    expect(last()).toEqual({
      delimiter: ';',
      has_header: true,
      custom_headers: [],
      trim_spaces: true,
      skip_empty_rows: true,
      input_key: '',
    });
    await user.type(screen.getByPlaceholderText(c.inputKeyPlaceholder), 'csv');
    expect(last().input_key).toBe('csv');
    await user.click(nodesField(c.trimSpaces));
    expect(last().trim_spaces).toBe(false);
    await user.click(nodesField(c.skipEmptyRows));
    expect(last().skip_empty_rows).toBe(false);
  });

  it('manages custom headers when the CSV has no header row', async () => {
    const { user, last } = renderConfig(CSVToJSONNodeConfigComponent, { has_header: true } as CSVToJSONNodeConfig);
    await user.click(nodesField(c.hasHeader));
    expect(last().has_header).toBe(false);
    expect(screen.getByText(c.noCustomHeaders)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: c.addHeader }));
    await user.click(screen.getByRole('button', { name: c.addHeader }));
    expect(last().custom_headers).toEqual(['', '']);

    const inputs = screen.getAllByPlaceholderText(c.headerPlaceholder);
    fireEvent.change(inputs[0], { target: { value: 'id' } });
    fireEvent.change(inputs[1], { target: { value: 'name' } });
    expect(last().custom_headers).toEqual(['id', 'name']);

    const removeButtons = screen.getAllByRole('button').filter((b) => b.textContent === '');
    await user.click(removeButtons[0]);
    expect(last().custom_headers).toEqual(['name']);
  });
});
