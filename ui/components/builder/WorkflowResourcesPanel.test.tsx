import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { WorkflowResourcesPanel } from './WorkflowResourcesPanel';
import { useDagStore } from '@/store/dagStore';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { resetBuilderStores, stubClipboard } from '@/test/builderFlowMocks';

const resource = (id: string, name: string) => ({ id, name, type: 'file_storage', description: '', status: 'active' });

let api: ApiMock;
beforeEach(() => {
  resetBuilderStores();
  api = mockApi();
  api.on('GET', '/resources', { data: { resources: [resource('r1', 'Main storage'), resource('r2', 'Archive')] } });
});
afterEach(() => api.restore());

const resources = () => useDagStore.getState().resources;
const attachedRow = (alias: string) => screen.getByText(alias).closest('.group') as HTMLElement;
const select = () => screen.getByRole('combobox');

describe('WorkflowResourcesPanel', () => {
  it('renders nothing when closed and does not load', () => {
    const { container } = render(<WorkflowResourcesPanel isOpen={false} onClose={() => {}} />);
    expect(container).toBeEmptyDOMElement();
    expect(api.calls).toHaveLength(0);
  });

  it('loads available resources and attaches one locally for an unsaved workflow', async () => {
    const onClose = vi.fn();
    render(<WorkflowResourcesPanel isOpen onClose={onClose} />);
    expect(screen.getByText('No resources attached yet')).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Main storage (file_storage)' })).toBeInTheDocument();

    const add = screen.getByRole('button', { name: 'Add' });
    expect(add).toBeDisabled();
    await userEvent.selectOptions(select(), 'r1');
    await userEvent.type(screen.getByPlaceholderText(/Alias/), ' main ');
    await userEvent.click(add);

    expect(resources()).toEqual([{ resource_id: 'r1', alias: 'main', access_type: 'read' }]);
    expect(screen.getByText('Attached Resources (1)')).toBeInTheDocument();
    expect(screen.getByText('(Main storage)')).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Main storage (file_storage)' })).not.toBeInTheDocument();
    expect(select()).toHaveValue('');

    await userEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalled();
  });

  it('attaches through the API for saved workflows and shows backend errors', async () => {
    useDagStore.setState({ dagId: 'wf1' });
    api.once('POST', '/workflows/wf1/resources', { status: 500, data: { message: 'boom' } });
    api.on('POST', '/workflows/wf1/resources', { data: {} });
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    await screen.findByRole('option', { name: 'Archive (file_storage)' });

    await userEvent.selectOptions(select(), 'r2');
    await userEvent.type(screen.getByPlaceholderText(/Alias/), 'arch');
    await userEvent.click(screen.getByRole('button', { name: 'Add' }));
    expect(await screen.findByText('Failed to attach resource. Please try again.')).toBeInTheDocument();
    expect(resources()).toEqual([]);

    await userEvent.click(screen.getByRole('button', { name: 'Add' }));
    await waitFor(() => expect(resources()).toHaveLength(1));
    expect(screen.queryByText('Failed to attach resource. Please try again.')).not.toBeInTheDocument();
    expect(api.last()?.data).toEqual({ resource_id: 'r2', alias: 'arch', access_type: 'read' });
  });

  it('shows validation errors and lets the user dismiss them', async () => {
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    await screen.findByRole('option', { name: 'Archive (file_storage)' });
    await userEvent.selectOptions(select(), 'r2');
    await userEvent.type(screen.getByPlaceholderText(/Alias/), '1bad');
    await userEvent.click(screen.getByRole('button', { name: 'Add' }));
    const message = await screen.findByText('Failed to attach resource. Please try again.');
    const banner = message.closest('div.mx-4') as HTMLElement;
    await userEvent.click(within(banner).getByRole('button'));
    expect(screen.queryByText('Failed to attach resource. Please try again.')).not.toBeInTheDocument();
  });

  it('shows an error when resources fail to load', async () => {
    api.once('GET', '/resources', { status: 500, data: {} });
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    expect(await screen.findByText('Failed to load resources. Please try again.')).toBeInTheDocument();
  });

  it('handles an empty resources response', async () => {
    api.once('GET', '/resources', { data: {} });
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    await waitFor(() => expect(select()).not.toBeDisabled());
    expect(screen.getAllByRole('option')).toHaveLength(1);
  });

  it('renames aliases, keeping unchanged or empty values', async () => {
    useDagStore.setState({ dagId: 'wf1', resources: [{ resource_id: 'r1', alias: 'main', access_type: 'read' }] });
    api.once('PUT', '/workflows/wf1/resources/r1', { status: 500, data: {} });
    api.on('PUT', '/workflows/wf1/resources/r1', { data: {} });
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    await screen.findByText('(Main storage)');

    // unchanged value just leaves edit mode
    await userEvent.click(screen.getByText('main'));
    fireEvent.blur(screen.getByDisplayValue('main'));
    expect(screen.queryByDisplayValue('main')).not.toBeInTheDocument();

    // failed update keeps edit mode and shows an error
    await userEvent.click(screen.getByText('main'));
    await userEvent.clear(screen.getByDisplayValue('main'));
    const input = document.querySelector('input.border-blue-500') as HTMLInputElement;
    await userEvent.type(input, 'primary{Enter}');
    expect(await screen.findByText('Failed to update alias. Please try again.')).toBeInTheDocument();
    expect(resources()[0].alias).toBe('main');

    await userEvent.type(input, '{Enter}');
    await waitFor(() => expect(resources()[0].alias).toBe('primary'));
    expect(screen.getByText('primary')).toBeInTheDocument();

    // empty value cancels
    await userEvent.click(screen.getByText('primary'));
    const again = document.querySelector('input.border-blue-500') as HTMLInputElement;
    await userEvent.clear(again);
    await userEvent.type(again, '{Enter}');
    expect(screen.getByText('primary')).toBeInTheDocument();
  });

  it('detaches resources, restoring them when the API fails', async () => {
    useDagStore.setState({ dagId: 'wf1', resources: [{ resource_id: 'r9', alias: 'gone', access_type: 'read' }] });
    api.once('DELETE', '/workflows/wf1/resources/r9', { status: 500, data: {} });
    api.on('DELETE', '/workflows/wf1/resources/r9', { data: {} });
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    expect(screen.getByText('(r9)')).toBeInTheDocument();

    await userEvent.click(within(attachedRow('gone')).getByTitle('Remove resource'));
    expect(await screen.findByText('Failed to detach resource. Please try again.')).toBeInTheDocument();
    expect(resources()).toHaveLength(1);

    await userEvent.click(within(attachedRow('gone')).getByTitle('Remove resource'));
    await waitFor(() => expect(resources()).toHaveLength(0));
    expect(screen.getByText('No resources attached yet')).toBeInTheDocument();
  });

  it('copies the resource template', async () => {
    userEvent.setup();
    const writeText = stubClipboard();
    useDagStore.setState({ resources: [{ resource_id: 'r1', alias: 'main', access_type: 'read' }] });
    render(<WorkflowResourcesPanel isOpen onClose={() => {}} />);
    await screen.findByText('(Main storage)');
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const copy = within(attachedRow('main')).getByTitle('Copy template');
      fireEvent.click(copy);
      expect(writeText).toHaveBeenCalledWith('{{resource.main}}');
      expect(copy.querySelector('.text-green-500')).not.toBeNull();
      act(() => vi.advanceTimersByTime(2100));
      expect(copy.querySelector('.text-green-500')).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });
});
