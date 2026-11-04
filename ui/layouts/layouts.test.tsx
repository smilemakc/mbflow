import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { BuilderLayout } from './BuilderLayout';
import { PageLayout } from './PageLayout';
import { useUIStore } from '@/store/uiStore';
import { useDagStore } from '@/store/dagStore';
import { makeUser, renderPage, signIn } from '@/test/pagesRouter';

// The builder's panels are heavy (React Flow, editors) and covered elsewhere;
// the layout's own job is composition, routing and keyboard shortcuts.
vi.mock('@/components/builder', () => ({
  Header: ({ onSave }: { onSave: () => void }) => <button onClick={onSave}>header save</button>,
}));
vi.mock('@/components/navigation', () => ({ Sidebar: () => <nav>builder sidebar</nav> }));
vi.mock('@/components/builder/NodeLibrary', () => ({ NodeLibrary: () => <div>node library</div> }));
vi.mock('@/components/builder/DAGCanvas', () => ({ DAGCanvas: () => <div>canvas</div> }));
vi.mock('@/components/builder/PropertiesPanel', () => ({ PropertiesPanel: () => <div>properties</div> }));
vi.mock('@/components/builder/MonitoringPanel', () => ({ MonitoringPanel: () => <div>monitoring</div> }));
vi.mock('@/components/builder/EdgeConfigPanel', () => ({ EdgeConfigPanel: () => <div>edge config</div> }));
vi.mock('@/components/modals/ShortcutsModal', () => ({ ShortcutsModal: () => <div>shortcuts modal</div> }));
vi.mock('@/components/modals/TemplatesModal', () => ({ TemplatesModal: () => <div>templates modal</div> }));
vi.mock('@/components/builder/WorkflowVariablesEditor', () => ({
  WorkflowVariablesEditor: ({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) =>
    isOpen ? <button onClick={onClose}>close variables</button> : null,
}));

describe('PageLayout', () => {
  beforeEach(() => {
    useUIStore.setState({ language: 'en', activeModal: null, isSidebarCollapsed: false });
    signIn(makeUser());
  });

  it('renders the sidebar, title and content', () => {
    renderPage(
      <PageLayout title="Triggers">
        <p>page body</p>
      </PageLayout>,
    );
    expect(screen.getByRole('heading', { level: 1, name: 'Triggers' })).toBeInTheDocument();
    expect(screen.getByText('page body')).toBeInTheDocument();
    expect(screen.queryByText('templates modal')).toBeNull();
    expect(screen.queryByText('shortcuts modal')).toBeNull();
  });

  it('falls back to the app name and shows the active modal', () => {
    useUIStore.setState({ activeModal: 'templates' });
    const view = renderPage(<PageLayout>body</PageLayout>);
    expect(screen.getByRole('heading', { level: 1, name: 'MBFlow' })).toBeInTheDocument();
    expect(screen.getByText('templates modal')).toBeInTheDocument();
    view.unmount();

    useUIStore.setState({ activeModal: 'shortcuts' });
    renderPage(<PageLayout>body</PageLayout>);
    expect(screen.getByText('shortcuts modal')).toBeInTheDocument();
  });
});

describe('BuilderLayout', () => {
  const actions = {
    saveDAG: vi.fn(),
    undo: vi.fn(),
    redo: vi.fn(),
    runWorkflow: vi.fn(),
    fetchWorkflow: vi.fn(),
    resetToNew: vi.fn(),
  };
  const original = useDagStore.getState();

  beforeEach(() => {
    Object.values(actions).forEach((fn) => fn.mockReset());
    actions.saveDAG.mockResolvedValue(undefined);
    useDagStore.setState(actions as never);
    useUIStore.setState({ activeModal: null, isFullscreen: false });
  });

  afterEach(() => {
    // Unmount before restoring the real actions so they are never invoked.
    cleanup();
    useDagStore.setState(original, true);
  });

  const renderBuilder = (at = '/builder') => renderPage(<BuilderLayout />, { path: '/builder/:workflowId?', at });
  const press = (key: string, init: KeyboardEventInit = {}, target: Window | Element = window) =>
    fireEvent.keyDown(target, { key, ...init });

  it('starts a new workflow without an id and loads one with an id', () => {
    const view = renderBuilder();
    expect(actions.resetToNew).toHaveBeenCalled();
    expect(actions.fetchWorkflow).not.toHaveBeenCalled();
    for (const part of ['builder sidebar', 'node library', 'canvas', 'monitoring', 'properties', 'edge config']) {
      expect(screen.getByText(part)).toBeInTheDocument();
    }
    view.unmount();

    renderBuilder('/builder/wf-42');
    expect(actions.fetchWorkflow).toHaveBeenCalledWith('wf-42');
  });

  it('redirects to the new workflow URL after the first save', async () => {
    actions.saveDAG.mockResolvedValue('wf-new');
    renderBuilder();
    fireEvent.click(screen.getByText('header save'));
    await waitFor(() => expect(actions.fetchWorkflow).toHaveBeenLastCalledWith('wf-new'));
    expect(screen.getByText('node library')).toBeInTheDocument();
  });

  it('stays on the page when saving an existing workflow', async () => {
    renderBuilder('/builder/wf-1');
    press('s', { ctrlKey: true });
    await waitFor(() => expect(actions.saveDAG).toHaveBeenCalled());
    expect(actions.fetchWorkflow).toHaveBeenCalledTimes(1);
  });

  it('maps keyboard shortcuts to editor actions', () => {
    renderBuilder('/builder/wf-1');

    press('Enter', { metaKey: true });
    expect(actions.runWorkflow).toHaveBeenCalledTimes(1);
    press('z', { ctrlKey: true });
    expect(actions.undo).toHaveBeenCalledTimes(1);
    press('z', { ctrlKey: true, shiftKey: true });
    press('y', { ctrlKey: true });
    expect(actions.redo).toHaveBeenCalledTimes(2);

    press('/', { ctrlKey: true });
    expect(screen.getByText('shortcuts modal')).toBeInTheDocument();
    press('/', { ctrlKey: true });
    expect(screen.queryByText('shortcuts modal')).toBeNull();

    press('F', { shiftKey: true });
    expect(useUIStore.getState().isFullscreen).toBe(true);
    expect(screen.queryByText('builder sidebar')).toBeNull();

    // Unbound keys do nothing.
    press('x', { ctrlKey: true });
    press('a');
    expect(actions.saveDAG).not.toHaveBeenCalled();
  });

  it('ignores shortcuts typed into form fields', () => {
    renderBuilder('/builder/wf-1');
    const input = document.createElement('input');
    const textarea = document.createElement('textarea');
    document.body.append(input, textarea);
    press('z', { ctrlKey: true }, input);
    press('Enter', { ctrlKey: true }, textarea);
    expect(actions.undo).not.toHaveBeenCalled();
    expect(actions.runWorkflow).not.toHaveBeenCalled();
    input.remove();
    textarea.remove();
  });

  it('shows the templates and variables modals', () => {
    useUIStore.setState({ activeModal: 'templates' });
    const view = renderBuilder('/builder/wf-1');
    expect(screen.getByText('templates modal')).toBeInTheDocument();
    view.unmount();

    useUIStore.setState({ activeModal: 'variables' });
    renderBuilder('/builder/wf-1');
    fireEvent.click(screen.getByText('close variables'));
    expect(useUIStore.getState().activeModal).toBeNull();
  });
});
