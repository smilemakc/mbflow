import { beforeEach, describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { FileStorageNodeConfigComponent } from './FileStorageNodeConfig';
import type { FileStorageNodeConfig } from '@/types/nodeConfigs';
import { useDagStore } from '@/store/dagStore';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('FileStorageNodeConfig', () => {
  const t = nodesEnT();
  const f = t.nodeConfig.fileStorage;

  beforeEach(() => {
    useDagStore.setState({
      resources: [
        { resource_id: 'r1', alias: 'files', resource_type: 'file_storage', resource_name: 'Main files' },
        { resource_id: 'r2', alias: 'cred', resource_type: 'credentials', resource_name: 'Secret' },
      ],
    } as any);
  });

  it('normalises an empty config to the store action defaults', () => {
    const { last } = renderConfig(FileStorageNodeConfigComponent, {} as FileStorageNodeConfig);
    expect(last()).toEqual({
      action: 'store',
      storage_id: '',
      file_source: 'url',
      file_data: '',
      file_url: '',
      file_name: '',
      mime_type: '',
      file_id: '',
      access_scope: 'workflow',
      ttl: 0,
      tags: [],
      limit: 100,
      offset: 0,
    });
    expect(screen.getByText(f.fileSource)).toBeInTheDocument();
    expect(screen.getByText(f.storageIdHint)).toBeInTheDocument();
  });

  it('edits store fields, switching between URL and base64 sources', async () => {
    const { user, last } = renderConfig(FileStorageNodeConfigComponent, { action: 'store' } as FileStorageNodeConfig);
    await user.type(screen.getByPlaceholderText(f.storageIdPlaceholder), 's1');
    expect(last().storage_id).toBe('s1');
    await user.type(screen.getByPlaceholderText(f.fileUrlPlaceholder), 'https://f');
    expect(last().file_url).toBe('https://f');
    await user.type(screen.getByPlaceholderText(f.fileNamePlaceholder), 'a.pdf');
    expect(last().file_name).toBe('a.pdf');
    await user.type(screen.getByPlaceholderText(f.mimeTypePlaceholder), 'application/pdf');
    expect(last().mime_type).toBe('application/pdf');

    await user.selectOptions(nodesField<HTMLSelectElement>(f.sourceType), 'base64');
    expect(last().file_source).toBe('base64');
    expect(screen.getByText(f.base64Data, { selector: 'label' })).toBeInTheDocument();
    await user.type(screen.getByPlaceholderText(f.base64Placeholder), 'QUJD');
    expect(last().file_data).toBe('QUJD');

    await user.selectOptions(nodesField<HTMLSelectElement>(f.accessScope), 'edge');
    expect(last().access_scope).toBe('edge');

    const ttl = screen.getByPlaceholderText(f.ttlPlaceholder);
    fireEvent.change(ttl, { target: { value: '3600' } });
    expect(last().ttl).toBe(3600);
    fireEvent.change(ttl, { target: { value: 'abc' } });
    expect(last().ttl).toBe(0);

    await user.type(screen.getByPlaceholderText(f.tagsPlaceholder), 'x, y');
    expect(last().tags).toEqual(['x', 'y']);
    await user.clear(screen.getByPlaceholderText(f.tagsPlaceholder));
    expect(last().tags).toEqual([]);
  });

  it('switches to the resource selector and back', async () => {
    const { user, last } = renderConfig(FileStorageNodeConfigComponent, { action: 'store' } as FileStorageNodeConfig);
    await user.click(screen.getByRole('button', { name: 'Use Resource' }));
    const select = screen.getByDisplayValue('Select file storage resource...');
    expect(screen.getByRole('option', { name: 'files (Main files)' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /cred/ })).toBeNull();
    await user.selectOptions(select, '{{resource.files}}');
    expect(last().storage_id).toBe('{{resource.files}}');
    await user.click(screen.getByRole('button', { name: 'Manual ID' }));
    expect(screen.getByPlaceholderText(f.storageIdPlaceholder)).toHaveValue('{{resource.files}}');
  });

  it('starts with the resource selector when the storage is a resource reference', () => {
    renderConfig(FileStorageNodeConfigComponent, { action: 'get', storage_id: '{{resource.files}}' } as FileStorageNodeConfig);
    expect(screen.getByRole('button', { name: 'Manual ID' })).toBeInTheDocument();
    expect(screen.getByDisplayValue('files (Main files)')).toBeInTheDocument();
  });

  it.each(['get', 'delete', 'metadata'] as const)('%s action asks for a file id only', async (action) => {
    const { user, last } = renderConfig(FileStorageNodeConfigComponent, { action } as FileStorageNodeConfig);
    expect(screen.queryByText(f.fileSource)).toBeNull();
    await user.type(screen.getByPlaceholderText(f.fileIdPlaceholder), 'id-1');
    expect(last()).toMatchObject({ action, file_id: 'id-1' });
  });

  it('list action exposes filters and pagination', async () => {
    const { user, last } = renderConfig(FileStorageNodeConfigComponent, { action: 'store' } as FileStorageNodeConfig);
    await user.selectOptions(nodesField<HTMLSelectElement>(f.action), 'list');
    expect(screen.getByText(f.filters)).toBeInTheDocument();

    await user.selectOptions(nodesField<HTMLSelectElement>(f.accessScope), '');
    expect(last().access_scope).toBe('');
    await user.selectOptions(nodesField<HTMLSelectElement>(f.accessScope), 'result');
    expect(last().access_scope).toBe('result');

    await user.type(nodesField(f.tagsFilter), 'tag1');
    expect(last().tags).toEqual(['tag1']);

    fireEvent.change(screen.getByPlaceholderText(f.limitPlaceholder), { target: { value: '20' } });
    expect(last().limit).toBe(20);
    fireEvent.change(screen.getByPlaceholderText(f.limitPlaceholder), { target: { value: '' } });
    expect(last().limit).toBe(100);
    fireEvent.change(screen.getByPlaceholderText(f.offsetPlaceholder), { target: { value: '40' } });
    expect(last().offset).toBe(40);
    fireEvent.change(screen.getByPlaceholderText(f.offsetPlaceholder), { target: { value: 'x' } });
    expect(last().offset).toBe(0);
  });

  it('resyncs fields and tags from the parent', () => {
    const { setConfig } = renderConfig(FileStorageNodeConfigComponent, { action: 'store' } as FileStorageNodeConfig);
    act(() => setConfig({ action: 'store', file_name: 'z.txt', tags: ['a', 'b'] }));
    expect(screen.getByPlaceholderText(f.fileNamePlaceholder)).toHaveValue('z.txt');
    expect(screen.getByPlaceholderText(f.tagsPlaceholder)).toHaveValue('a, b');
  });
});
