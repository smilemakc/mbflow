import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Header } from './Header';
import { useDagStore } from '@/store/dagStore';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { makeNode, resetBuilderStores } from '@/test/builderFlowMocks';

const wf = (id: string, name: string) => ({
  id, name, description: '', status: 'draft', version: 1, variables: {}, metadata: {}, nodes: [], edges: [], created_at: '', updated_at: '',
});

const s = () => useDagStore.getState();
const ui = () => useUIStore.getState();
const toasts = () => useToastStore.getState().toasts;

let api: ApiMock;
beforeEach(() => {
  resetBuilderStores();
  api = mockApi();
  api.on('GET', '/resources', { data: { resources: [] } });
});
afterEach(() => api.restore());

describe('Header', () => {
  it('renames the workflow with Enter, the check button and blur', async () => {
    useDagStore.setState({ dagName: 'Flow' });
    render(<Header />);
    await userEvent.click(screen.getByTitle('Click to edit workflow name'));
    const input = screen.getByDisplayValue('Flow');
    expect(input).toHaveFocus();
    await userEvent.clear(input);
    await userEvent.type(input, '  Billing  {Enter}');
    expect(s().dagName).toBe('Billing');
    expect(s().isDirty).toBe(true);

    await userEvent.click(screen.getByTitle('Click to edit workflow name'));
    await userEvent.type(screen.getByDisplayValue('Billing'), ' v2');
    await userEvent.click(screen.getByTitle('Save'));
    expect(s().dagName).toBe('Billing v2');

    await userEvent.click(screen.getByTitle('Click to edit workflow name'));
    await userEvent.type(screen.getByDisplayValue('Billing v2'), '!');
    await userEvent.click(document.body);
    expect(s().dagName).toBe('Billing v2!');
  });

  it('cancels renaming with Escape or the cancel button and restores empty names', async () => {
    useDagStore.setState({ dagName: 'Flow' });
    render(<Header />);
    await userEvent.click(screen.getByTitle('Click to edit workflow name'));
    await userEvent.type(screen.getByDisplayValue('Flow'), 'xx{Escape}');
    expect(screen.getByText('Flow')).toBeInTheDocument();

    await userEvent.click(screen.getByTitle('Click to edit workflow name'));
    await userEvent.type(screen.getByDisplayValue('Flow'), 'yy');
    await userEvent.click(screen.getByTitle('Cancel'));
    expect(s().dagName).toBe('Flow');

    await userEvent.click(screen.getByTitle('Click to edit workflow name'));
    await userEvent.clear(screen.getByDisplayValue('Flow'));
    await userEvent.keyboard('{Enter}');
    expect(s().dagName).toBe('Flow');
    expect(screen.getByText('Flow')).toBeInTheDocument();

    useDagStore.setState({ dagName: '' });
    expect(await screen.findByText('Untitled Workflow')).toBeInTheDocument();
    useDagStore.setState({ dagName: 'External' });
    expect(await screen.findByText('External')).toBeInTheDocument();
  });

  it('opens modals and toggles UI preferences', async () => {
    render(<Header />);
    await userEvent.click(screen.getByTitle('Workflow Variables'));
    expect(ui().activeModal).toBe('variables');
    await userEvent.click(screen.getByTitle('Templates'));
    expect(ui().activeModal).toBe('templates');
    await userEvent.click(screen.getByTitle('Shortcuts'));
    expect(ui().activeModal).toBe('shortcuts');

    await userEvent.click(screen.getByRole('button', { name: 'Components' }));
    expect(ui().isNodeLibraryOpen).toBe(true);
    await userEvent.click(screen.getByTitle('Switch Language'));
    expect(ui().language).toBe('ru');
    useUIStore.setState({ language: 'en' });
    await userEvent.click(await screen.findByTitle('Focus Mode'));
    expect(ui().isFullscreen).toBe(true);
    await userEvent.click(screen.getByTitle('Focus Mode'));
    expect(ui().isFullscreen).toBe(false);
    await userEvent.click(screen.getByTitle('Toggle Theme'));
    expect(ui().theme).toBe('dark');
    await userEvent.click(screen.getByTitle('Toggle Theme'));
    expect(ui().theme).toBe('light');
  });

  it('opens and closes the resources panel', async () => {
    render(<Header />);
    await userEvent.click(screen.getByTitle('Workflow Resources'));
    expect(await screen.findByText('Manage resources attached to this workflow')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(screen.queryByText('Manage resources attached to this workflow')).not.toBeInTheDocument();
  });

  it('undoes and redoes graph changes', async () => {
    render(<Header />);
    const undo = screen.getByTitle('Undo');
    const redo = screen.getByTitle('Redo');
    expect(undo).toBeDisabled();
    expect(redo).toBeDisabled();
    s().addNode(makeNode('x').data.type, { x: 0, y: 0 });
    await waitFor(() => expect(undo).toBeEnabled());
    await userEvent.click(undo);
    expect(s().nodes).toHaveLength(0);
    expect(redo).toBeEnabled();
    await userEvent.click(redo);
    expect(s().nodes).toHaveLength(1);
  });

  it('saves a new workflow through the API and shows the saved state', async () => {
    api.on('POST', '/workflows', (c) => ({ data: wf('new-id', (c.data as { name: string }).name) }));
    api.on('PUT', '/workflows/new-id', (c) => ({ data: { ...wf('new-id', 'Flow'), ...(c.data as object) } }));
    useDagStore.setState({ dagName: 'Flow', isDirty: true, nodes: [makeNode('a')] });
    render(<Header />);
    expect(screen.getByText('Unsaved')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(s().dagId).toBe('new-id'));
    expect(screen.getByText('Saved')).toBeInTheDocument();
    expect(api.calls.map((c) => `${c.method} ${c.url}`)).toEqual(['POST /workflows', 'PUT /workflows/new-id']);
    expect(api.last()?.data).toMatchObject({ name: 'Flow' });
  });

  it('updates an existing workflow and reports save errors as toasts', async () => {
    api.once('PUT', '/workflows/w1', { status: 500, data: { message: 'disk full' } });
    api.on('PUT', '/workflows/w1', { data: wf('w1', 'Flow') });
    useDagStore.setState({ dagId: 'w1', dagName: 'Flow', isDirty: true });
    render(<Header />);
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(toasts()).toHaveLength(1));
    expect(toasts()[0]).toMatchObject({ type: 'error', title: 'Save Failed' });
    expect(s().isDirty).toBe(true);
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(s().isDirty).toBe(false));
  });

  it('shows the saving indicator and uses a custom save handler', async () => {
    const onSave = vi.fn();
    useDagStore.setState({ isSaving: true });
    render(<Header onSave={onSave} />);
    expect(screen.getByText('Saving...')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSave).toHaveBeenCalled();
    expect(api.calls.filter((c) => c.url.startsWith('/workflows'))).toHaveLength(0);
  });

  it('runs the workflow and disables actions while running', async () => {
    const runWorkflow = vi.fn();
    useDagStore.setState({ runWorkflow });
    render(<Header />);
    await userEvent.click(screen.getByRole('button', { name: 'Run' }));
    expect(runWorkflow).toHaveBeenCalled();

    useDagStore.setState({ isRunning: true, history: [{ nodes: [], edges: [] }, { nodes: [], edges: [] }], historyIndex: 1 });
    expect(await screen.findByRole('button', { name: 'Running...' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
    expect(screen.getByTitle('Undo')).toBeDisabled();
  });

  it('shows an error toast when an unsaved workflow cannot be saved before running', async () => {
    api.on('POST', '/workflows', { status: 500, data: { message: 'down' } });
    render(<Header />);
    await userEvent.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(toasts().map((t) => t.title)).toEqual(['Save Failed', 'Cannot Run']));
    expect(s().isRunning).toBe(false);
  });
});
