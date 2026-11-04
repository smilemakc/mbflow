import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { GoogleDriveNodeConfigComponent } from './GoogleDriveNodeConfig';
import type { GoogleDriveNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

const ops = (c: Partial<GoogleDriveNodeConfig>) => ({ credentials: '', ...c }) as GoogleDriveNodeConfig;

describe('GoogleDriveNodeConfig', () => {
  const t = nodesEnT();
  const g = t.nodeConfig.googleDrive;

  it('list_files (default) shows parent folder and list options', () => {
    render(<GoogleDriveNodeConfigComponent config={undefined as unknown as GoogleDriveNodeConfig} onChange={() => {}} />);
    expect(nodesField<HTMLSelectElement>(g.operation)).toHaveValue('list_files');
    expect(screen.getByPlaceholderText(g.parentFolderIdPlaceholder)).toBeInTheDocument();
    expect(nodesField(g.maxResults)).toHaveValue(100);
    expect(nodesField<HTMLSelectElement>(g.orderBy)).toHaveValue('modifiedTime desc');
    expect(screen.queryByPlaceholderText(g.fileIdPlaceholder)).toBeNull();
    expect(screen.getByText('Returns: success, files (array), file_count')).toBeInTheDocument();
  });

  it('edits list options, credentials and parent folder', async () => {
    const { user, last } = renderConfig(GoogleDriveNodeConfigComponent, ops({ operation: 'list_files' }));
    await user.type(screen.getByPlaceholderText(g.credentialsPlaceholder), 'k');
    expect(last()).toMatchObject({ operation: 'list_files', credentials: 'k', max_results: 100, order_by: 'modifiedTime desc' });
    await user.type(screen.getByPlaceholderText(g.parentFolderIdPlaceholder), 'p');
    expect(last().parent_folder_id).toBe('p');
    fireEvent.change(nodesField(g.maxResults), { target: { value: '25' } });
    expect(last().max_results).toBe(25);
    fireEvent.change(nodesField(g.maxResults), { target: { value: '' } });
    expect(last().max_results).toBe(100);
    await user.selectOptions(nodesField<HTMLSelectElement>(g.orderBy), 'name');
    expect(last().order_by).toBe('name');
  });

  it.each([
    ['create_spreadsheet', { fileName: true, folderName: false, fileId: false, dest: false, parent: true }, 'Returns: success, file_id, file_name, web_view_url'],
    ['create_folder', { fileName: false, folderName: true, fileId: false, dest: false, parent: true }, 'Returns: success, folder_id, file_name, web_view_url'],
    ['delete', { fileName: false, folderName: false, fileId: true, dest: false, parent: false }, 'Returns: success, file_id, file_name'],
    ['move', { fileName: false, folderName: false, fileId: true, dest: true, parent: false }, 'Returns: success, file_id, file_name, web_view_url'],
    ['copy', { fileName: true, folderName: false, fileId: true, dest: true, parent: false }, 'Returns: success, file_id, file_name, source_file_id'],
  ] as const)('%s shows only its fields', (operation, shown, returns) => {
    renderConfig(GoogleDriveNodeConfigComponent, ops({ operation }));
    const has = (text: string) => screen.queryByText(text, { selector: 'span' }) !== null;
    expect(has(g.fileNameHint)).toBe(shown.fileName);
    expect(screen.queryByPlaceholderText(g.folderNamePlaceholder) !== null).toBe(shown.folderName);
    expect(screen.queryByPlaceholderText(g.fileIdPlaceholder) !== null).toBe(shown.fileId);
    expect(screen.queryByPlaceholderText(g.destinationFolderIdPlaceholder) !== null).toBe(shown.dest);
    expect(screen.queryByPlaceholderText(g.parentFolderIdPlaceholder) !== null).toBe(shown.parent);
    expect(screen.queryByText(g.maxResults)).toBeNull();
    expect(screen.getByText(returns)).toBeInTheDocument();
  });

  it('emits file, folder and destination fields', async () => {
    const { user, last } = renderConfig(GoogleDriveNodeConfigComponent, ops({ operation: 'copy' }));
    expect(screen.getByPlaceholderText('Copy of Original File')).toBeInTheDocument();
    await user.type(screen.getByPlaceholderText('Copy of Original File'), 'n');
    expect(last().file_name).toBe('n');
    await user.type(screen.getByPlaceholderText(g.fileIdPlaceholder), 'f');
    expect(last().file_id).toBe('f');
    await user.type(screen.getByPlaceholderText(g.destinationFolderIdPlaceholder), 'd');
    expect(last().destination_folder_id).toBe('d');

    await user.selectOptions(nodesField<HTMLSelectElement>(g.operation), 'create_folder');
    await user.type(screen.getByPlaceholderText(g.folderNamePlaceholder), 'F');
    expect(last()).toMatchObject({ operation: 'create_folder', folder_name: 'F', file_name: 'n' });

    await user.selectOptions(nodesField<HTMLSelectElement>(g.operation), 'create_spreadsheet');
    expect(screen.getByPlaceholderText(g.fileNamePlaceholder)).toHaveValue('n');
    expect(screen.getAllByText(g.fileNameOptional).length).toBeGreaterThan(0);
  });
});
