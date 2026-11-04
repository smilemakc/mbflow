import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { WorkflowsPage } from './WorkflowsPage';
import { workflowService } from '@/services/workflowService';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { apiError } from '@/test/apiMock';
import { renderPage } from '@/test/pagesRouter';
import type { DAG } from '@/types';

const daysAgo = (d: number) => new Date(Date.now() - d * 24 * 60 * 60 * 1000).toISOString();

const wf = (id: string, overrides: Partial<DAG> = {}): DAG =>
  ({
    id,
    name: `Workflow ${id}`,
    description: '',
    status: 'draft',
    nodes: [],
    edges: [],
    createdAt: daysAgo(30),
    updatedAt: daysAgo(0),
    ...overrides,
  }) as DAG;

const WORKFLOWS: DAG[] = [
  wf('a', { name: 'Billing sync', description: 'Syncs invoices', status: 'active', nodes: [{ id: 'n1' }] as never, updatedAt: daysAgo(1) }),
  wf('b', { name: 'Email digest', status: 'inactive', nodes: [{ id: 'n1' }, { id: 'n2' }] as never, edges: [{ id: 'e1' }] as never, updatedAt: daysAgo(3) }),
  wf('c', { name: 'Old cleanup', status: 'archived', updatedAt: '2024-03-05T12:00:00Z' }),
];

const summary = () =>
  screen.getByText((_, el) => el?.tagName === 'P' && !!el.textContent?.startsWith('Showing'));
const toasts = () => useToastStore.getState().toasts;
const cardFor = (name: string) => screen.getByRole('heading', { name }).closest('div.rounded-xl') as HTMLElement;

describe('WorkflowsPage', () => {
  let getAll: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useToastStore.getState().clearAll();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    getAll = vi.spyOn(workflowService, 'getAll').mockResolvedValue(WORKFLOWS as never);
  });

  it('lists workflows with status, node counts and relative dates', async () => {
    renderPage(<WorkflowsPage />);
    expect(await screen.findByRole('heading', { name: 'Billing sync' })).toBeInTheDocument();

    const billing = cardFor('Billing sync');
    expect(within(billing).getByText('Active')).toBeInTheDocument();
    expect(within(billing).getByText('Syncs invoices')).toBeInTheDocument();
    expect(within(billing).getByText('node')).toBeInTheDocument();
    expect(within(billing).getByText('Updated Yesterday')).toBeInTheDocument();
    expect(within(billing).getByText(/^Created [A-Z][a-z]{2} \d+, \d{4}$/)).toBeInTheDocument();

    const digest = cardFor('Email digest');
    expect(within(digest).getByText('nodes')).toBeInTheDocument();
    expect(within(digest).getByText('Updated 3 days ago')).toBeInTheDocument();

    expect(within(cardFor('Old cleanup')).getByText('Updated Mar 5, 2024')).toBeInTheDocument();
    expect(within(cardFor('Old cleanup')).getByText('Archived')).toBeInTheDocument();

    expect(summary()).toHaveTextContent('Showing 3 workflows');
  });

  it('filters by search text and status', async () => {
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);
    await screen.findByRole('heading', { name: 'Billing sync' });

    await user.type(screen.getByPlaceholderText(/Search workflows/), 'invoices');
    expect(summary()).toHaveTextContent('Showing 1 workflow matching "invoices"');
    expect(screen.queryByRole('heading', { name: 'Email digest' })).toBeNull();

    await user.clear(screen.getByPlaceholderText(/Search workflows/));
    await user.selectOptions(screen.getByRole('combobox'), 'inactive');
    expect(screen.getByRole('heading', { name: 'Email digest' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Billing sync' })).toBeNull();

    await user.selectOptions(screen.getByRole('combobox'), 'draft');
    expect(screen.getByText('No workflows found')).toBeInTheDocument();
    expect(screen.getByText('Try adjusting your search or filter criteria.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Create Your First Workflow' })).toBeNull();
  });

  it('shows the empty state with a create action that opens the builder', async () => {
    getAll.mockResolvedValue([]);
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);

    expect(await screen.findByText('No workflows yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Create Your First Workflow' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/builder');
  });

  it('navigates to the builder for new and existing workflows', async () => {
    const user = userEvent.setup();
    const view = renderPage(<WorkflowsPage />);
    await screen.findByRole('heading', { name: 'Billing sync' });

    await user.click(within(cardFor('Billing sync')).getByRole('button', { name: 'Edit' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/builder/a');
    view.unmount();

    renderPage(<WorkflowsPage />);
    await user.click(await screen.findByRole('button', { name: 'Create New Workflow' }));
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/builder$/);
  });

  it('toggles the variables guide', async () => {
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);
    await screen.findByRole('heading', { name: 'Billing sync' });

    await user.click(screen.getByRole('button', { name: 'Variables Guide' }));
    expect(screen.getByText('Workflow Variables & Templates')).toBeInTheDocument();
    await user.click(screen.getByTitle('Close'));
    expect(screen.queryByText('Workflow Variables & Templates')).toBeNull();

    await user.click(screen.getByRole('button', { name: 'Variables Guide' }));
    await user.click(screen.getByRole('button', { name: 'Variables Guide' }));
    expect(screen.queryByText('Workflow Variables & Templates')).toBeNull();
  });

  it('paginates more than 12 workflows', async () => {
    const many = Array.from({ length: 14 }, (_, i) => wf(String(i + 1).padStart(2, '0')));
    getAll.mockResolvedValue(many);
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);

    expect(await screen.findByRole('heading', { name: 'Workflow 01' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Workflow 13' })).toBeNull();
    expect(within(cardFor('Workflow 01')).getByText('Updated Today')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Next page' }));
    expect(screen.getByRole('heading', { name: 'Workflow 13' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Workflow 01' })).toBeNull();
  });

  it('shows a load error and retries', async () => {
    getAll.mockRejectedValueOnce(apiError(500));
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);

    expect(await screen.findByText('Failed to load workflows. Please try again.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try Again' }));
    expect(await screen.findByRole('heading', { name: 'Billing sync' })).toBeInTheDocument();
    expect(getAll).toHaveBeenCalledTimes(2);
  });

  it('clones a workflow with its graph and reloads', async () => {
    const create = vi.spyOn(workflowService, 'create').mockResolvedValue(wf('copy', { name: 'Email digest (Copy)' }) as never);
    const save = vi.spyOn(workflowService, 'save').mockResolvedValue(wf('copy') as never);
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);
    await screen.findByRole('heading', { name: 'Email digest' });

    await user.click(within(cardFor('Email digest')).getByTitle('Clone workflow'));

    await waitFor(() => expect(getAll).toHaveBeenCalledTimes(2));
    expect(create).toHaveBeenCalledWith('Email digest (Copy)', '');
    expect(save).toHaveBeenCalledWith(expect.objectContaining({ id: 'copy', nodes: WORKFLOWS[1].nodes, edges: WORKFLOWS[1].edges }));
  });

  it('reports clone failures', async () => {
    vi.spyOn(workflowService, 'create').mockRejectedValue(apiError(500));
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);
    await screen.findByRole('heading', { name: 'Old cleanup' });

    await user.click(within(cardFor('Old cleanup')).getByTitle('Clone workflow'));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Clone Failed' }));
  });

  it('deletes a workflow after confirmation and reports failures', async () => {
    const remove = vi
      .spyOn(workflowService, 'delete')
      .mockResolvedValueOnce({} as never)
      .mockRejectedValueOnce(apiError(500));
    const user = userEvent.setup();
    renderPage(<WorkflowsPage />);
    await screen.findByRole('heading', { name: 'Billing sync' });

    await user.click(within(cardFor('Billing sync')).getByTitle('Delete workflow'));
    expect(screen.getByText('Delete Workflow')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(remove).not.toHaveBeenCalled();
    expect(screen.queryByText('Delete Workflow')).toBeNull();

    await user.click(within(cardFor('Billing sync')).getByTitle('Delete workflow'));
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(remove).toHaveBeenCalledWith('a');
    await waitFor(() => expect(getAll).toHaveBeenCalledTimes(2));

    await user.click(within(cardFor('Billing sync')).getByTitle('Delete workflow'));
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Delete Failed' }));
  });
});
