import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FilesModal } from './FilesModal';
import { useUIStore } from '@/store/uiStore';
import { fileMeta, storage } from '@/test/resourcesFixtures';

type Props = React.ComponentProps<typeof FilesModal>;

const setup = (overrides: Partial<Props> = {}) => {
  const props: Props = {
    isOpen: true,
    onClose: vi.fn(),
    resource: storage({ name: 'Docs' }),
    files: [],
    filesTotal: 0,
    loading: false,
    onUpload: vi.fn().mockResolvedValue(true),
    onDelete: vi.fn().mockResolvedValue(true),
    onDownload: vi.fn(),
    ...overrides,
  };
  const utils = render(<FilesModal {...props} />);
  return { ...utils, props, user: userEvent.setup() };
};

const fileInput = () => document.getElementById('file-upload') as HTMLInputElement;

describe('FilesModal', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('titles the modal with the resource name and shows the empty state', () => {
    setup();
    expect(screen.getByRole('heading', { name: 'Files - Docs' })).toBeInTheDocument();
    expect(screen.getByText('No files uploaded yet')).toBeInTheDocument();
    expect(screen.getByText('Choose file to upload')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Upload' })).not.toBeInTheDocument();
  });

  it('handles a missing resource in the title', () => {
    setup({ resource: null });
    expect(screen.getByRole('heading', { name: 'Files -' })).toBeInTheDocument();
  });

  it('shows a spinner while files load', () => {
    setup({ loading: true, files: [fileMeta()] });
    expect(screen.queryByText('report.pdf')).not.toBeInTheDocument();
    expect(screen.queryByText('No files uploaded yet')).not.toBeInTheDocument();
    expect(document.querySelector('.animate-spin')).not.toBeNull();
  });

  it('lists files with counts and total size, and downloads by id', async () => {
    const files = [fileMeta({ id: 'f1', name: 'a.txt', size: 1024 }), fileMeta({ id: 'f2', name: 'b.txt', size: 1024 })];
    const { props, user } = setup({ files, filesTotal: 10 });
    expect(screen.getByText('Showing 2 of 10 files')).toBeInTheDocument();
    expect(screen.getByText('Total: 2 KB')).toBeInTheDocument();
    expect(screen.getAllByText(/^1 KB •/)).toHaveLength(2);

    await user.click(screen.getByRole('button', { name: 'Download b.txt' }));
    expect(props.onDownload).toHaveBeenCalledWith('f2');
  });

  it('uploads the selected file and clears the selection on success', async () => {
    let resolve!: (v: boolean) => void;
    const onUpload = vi.fn(() => new Promise<boolean>((r) => (resolve = r)));
    const { user } = setup({ onUpload, files: [fileMeta({ id: 'f1', name: 'a.txt' })] });
    const file = new File(['hello world'], 'hello.txt', { type: 'text/plain' });

    await user.upload(fileInput(), file);
    expect(screen.getByText('hello.txt')).toBeInTheDocument();
    expect(screen.getByText('File size: 11 B')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Upload' }));
    expect(onUpload).toHaveBeenCalledWith(file);
    // Delete buttons are disabled during the upload.
    expect(screen.getByRole('button', { name: 'Delete a.txt' })).toBeDisabled();

    resolve(true);
    await waitFor(() => expect(screen.getByText('Choose file to upload')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: 'Upload' })).not.toBeInTheDocument();
  });

  it('keeps the selected file when the upload fails', async () => {
    const { user } = setup({ onUpload: vi.fn().mockResolvedValue(false) });
    await user.upload(fileInput(), new File(['x'], 'keep.bin'));
    await user.click(screen.getByRole('button', { name: 'Upload' }));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Upload' })).toBeEnabled());
    expect(screen.getByText('keep.bin')).toBeInTheDocument();
  });

  it('confirms before deleting a file', async () => {
    const { props, user } = setup({ files: [fileMeta({ id: 'f9', name: 'old.log' })], filesTotal: 1 });

    await user.click(screen.getByRole('button', { name: 'Delete old.log' }));
    const dialogs = screen.getAllByRole('dialog');
    const confirm = dialogs[dialogs.length - 1];
    expect(within(confirm).getByText('Delete File')).toBeInTheDocument();
    expect(within(confirm).getByText(/delete "old\.log"\?/)).toBeInTheDocument();

    await user.click(within(confirm).getByRole('button', { name: 'Cancel' }));
    expect(props.onDelete).not.toHaveBeenCalled();
    expect(screen.queryByText('Delete File')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Delete old.log' }));
    const again = screen.getAllByRole('dialog').pop()!;
    await user.click(within(again).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(props.onDelete).toHaveBeenCalledWith('f9', 'old.log'));
    await waitFor(() => expect(screen.queryByText('Delete File')).not.toBeInTheDocument());
  });

  it('clears the pending upload when closed', async () => {
    const { props, user, rerender } = setup();
    await user.upload(fileInput(), new File(['x'], 'pending.txt'));
    await user.click(screen.getByRole('button', { name: 'Close modal' }));
    expect(props.onClose).toHaveBeenCalledTimes(1);
    rerender(<FilesModal {...props} isOpen />);
    expect(screen.getByText('Choose file to upload')).toBeInTheDocument();
  });
});
