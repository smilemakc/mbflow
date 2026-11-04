import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { GoogleSheetsNodeConfigComponent } from './GoogleSheetsNodeConfig';
import type { GoogleSheetsNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('GoogleSheetsNodeConfig', () => {
  const t = nodesEnT();
  const g = t.nodeConfig.googleSheets;

  it('read operation hides write options and shows the read hint', () => {
    render(<GoogleSheetsNodeConfigComponent config={undefined as unknown as GoogleSheetsNodeConfig} onChange={() => {}} />);
    expect(nodesField<HTMLSelectElement>(g.operation)).toHaveValue('read');
    expect(screen.queryByText(g.valueInputOption)).toBeNull();
    expect(screen.getByText(/no input data is needed/)).toBeInTheDocument();
  });

  it('emits edits of the common fields', async () => {
    const { user, last } = renderConfig(GoogleSheetsNodeConfigComponent, { operation: 'read' } as GoogleSheetsNodeConfig);
    await user.type(screen.getByPlaceholderText(g.spreadsheetIdPlaceholder), 'sid');
    expect(last()).toMatchObject({ operation: 'read', spreadsheet_id: 'sid', value_input_option: 'USER_ENTERED', major_dimension: 'ROWS' });
    await user.type(screen.getByPlaceholderText(g.sheetNamePlaceholder), 'S1');
    expect(last().sheet_name).toBe('S1');
    await user.type(screen.getByPlaceholderText(g.rangePlaceholder), 'A1');
    expect(last().range).toBe('A1');
    await user.type(screen.getByPlaceholderText(g.credentialsPlaceholder), 'c');
    expect(last().credentials).toBe('c');
  });

  it('write/append operations expose value input, dimension and column mapping', async () => {
    const { user, last } = renderConfig(GoogleSheetsNodeConfigComponent, { operation: 'read' } as GoogleSheetsNodeConfig);
    await user.selectOptions(nodesField<HTMLSelectElement>(g.operation), 'append');
    expect(last().operation).toBe('append');
    expect(screen.getByText(/Supported formats/)).toBeInTheDocument();

    await user.selectOptions(nodesField<HTMLSelectElement>(g.valueInputOption), 'RAW');
    expect(last().value_input_option).toBe('RAW');
    await user.selectOptions(nodesField<HTMLSelectElement>(g.majorDimension), 'COLUMNS');
    expect(last().major_dimension).toBe('COLUMNS');
    await user.type(screen.getByPlaceholderText(g.columnMappingPlaceholder), 'title,link');
    expect(last().columns).toBe('title,link');

    await user.selectOptions(nodesField<HTMLSelectElement>(g.operation), 'write');
    expect(screen.getByText(g.valueInputOption)).toBeInTheDocument();
  });
});
