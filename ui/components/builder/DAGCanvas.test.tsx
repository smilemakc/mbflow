import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, createEvent, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReactFlowProvider } from 'reactflow';
import { DAGCanvas } from './DAGCanvas';
import { NODE_TEMPLATE_MIME, NODE_TYPE_MIME } from './dragData';
import { useDagStore } from '@/store/dagStore';
import { useUIStore } from '@/store/uiStore';
import { NodeType } from '@/types';
import { makeEdge, makeNode, resetBuilderStores, stubClipboard } from '@/test/builderFlowMocks';

const s = () => useDagStore.getState();

beforeEach(resetBuilderStores);

const renderCanvas = async (withGraph = true) => {
  if (withGraph) useDagStore.setState({ nodes: [makeNode('a'), makeNode('b')], edges: [makeEdge('e1', 'a', 'b')] });
  const utils = render(
    <ReactFlowProvider>
      <DAGCanvas />
    </ReactFlowProvider>,
  );
  // Wait for the reactflow instance (onInit) to be ready.
  await act(async () => {
    await new Promise((r) => setTimeout(r, 0));
  });
  const q = (sel: string) => utils.container.querySelector(sel) as HTMLElement;
  return { ...utils, q, pane: () => q('.react-flow__pane'), wrapper: () => q('.react-flow') };
};

const dropData = (values: Record<string, string>) => ({
  getData: (k: string) => values[k] ?? '',
  dropEffect: '',
});

/** jsdom has no DragEvent, so pointer coordinates are attached by hand. */
const drop = (target: HTMLElement, values: Record<string, string>, clientX = 0, clientY = 0) => {
  const event = createEvent.drop(target, { dataTransfer: dropData(values) });
  Object.defineProperties(event, { clientX: { value: clientX }, clientY: { value: clientY } });
  fireEvent(target, event);
};

describe('DAGCanvas', () => {
  it('renders nodes and edges from the store', async () => {
    const { container } = await renderCanvas();
    expect(screen.getByText('Label a')).toBeInTheDocument();
    expect(screen.getByText('Label b')).toBeInTheDocument();
    expect(container.querySelectorAll('.react-flow__edge')).toHaveLength(1);
  });

  it('selects nodes and edges on click and clears selection on pane click', async () => {
    const { q, pane } = await renderCanvas();
    fireEvent.click(q('[data-testid="rf__node-a"]'));
    expect(s().selectedNodeId).toBe('a');

    fireEvent.click(q('[data-testid="rf__edge-e1"]'));
    expect(s().selectedEdgeId).toBe('e1');
    expect(s().selectedNodeId).toBeNull();

    fireEvent.click(pane());
    expect(s().selectedEdgeId).toBeNull();
    expect(s().selectedNodeId).toBeNull();
  });

  it('adds nodes dropped from the palette, with templates', async () => {
    const { wrapper } = await renderCanvas(false);
    const over = dropData({});
    fireEvent.dragOver(wrapper(), { dataTransfer: over });
    expect(over.dropEffect).toBe('move');

    drop(wrapper(), {}, 10, 10);
    expect(s().nodes).toHaveLength(0);

    drop(wrapper(), { [NODE_TYPE_MIME]: NodeType.LLM }, 100, 150);
    expect(s().nodes).toHaveLength(1);
    expect(s().nodes[0]).toMatchObject({ id: 'llm', position: { x: 100, y: 150 } });
    expect(await screen.findByText('llm')).toBeInTheDocument();

    drop(wrapper(), {
      [NODE_TYPE_MIME]: NodeType.OPENAPI,
      [NODE_TEMPLATE_MIME]: JSON.stringify({ label: 'List pets', config: { client_id: 'c', operation_id: 'o' } }),
    });
    expect(s().nodes[1].data).toMatchObject({ label: 'List pets', type: NodeType.OPENAPI, config: { client_id: 'c', operation_id: 'o' } });
  });

  it('runs node context menu actions', async () => {
    const user = userEvent.setup();
    const writeText = stubClipboard();
    const { q } = await renderCanvas();

    fireEvent.contextMenu(q('[data-testid="rf__node-a"]'), { clientX: 30, clientY: 40 });
    await user.click(screen.getByRole('button', { name: 'Duplicate' }));
    expect(s().nodes).toHaveLength(3);
    expect(screen.queryByRole('button', { name: 'Duplicate' })).not.toBeInTheDocument();

    fireEvent.contextMenu(q('[data-testid="rf__node-b"]'));
    await user.click(screen.getByRole('button', { name: 'Copy ID' }));
    expect(writeText).toHaveBeenCalledWith('b');

    fireEvent.contextMenu(q('[data-testid="rf__node-b"]'));
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(s().nodes.map((n) => n.id)).not.toContain('b');
    expect(s().edges).toHaveLength(0);
  });

  it('opens the pane context menu, resets the view and closes on pane click', async () => {
    const { pane } = await renderCanvas();
    fireEvent.contextMenu(pane(), { clientX: 5, clientY: 5 });
    await userEvent.click(screen.getByRole('button', { name: 'Reset View' }));
    expect(screen.queryByRole('button', { name: 'Reset View' })).not.toBeInTheDocument();

    fireEvent.contextMenu(pane());
    expect(screen.getByRole('button', { name: 'Reset View' })).toBeInTheDocument();
    fireEvent.click(pane());
    expect(screen.queryByRole('button', { name: 'Reset View' })).not.toBeInTheDocument();
  });

  it('applies the auto layout in both directions', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      await renderCanvas();
      const before = s().nodes.map((n) => n.position);
      fireEvent.click(screen.getByTitle('Auto Layout (Top to Bottom)'));
      const tb = s().nodes.map((n) => n.position);
      expect(tb).not.toEqual(before);
      expect(tb[1].y).toBeGreaterThan(tb[0].y);

      fireEvent.click(screen.getByTitle('Auto Layout (Left to Right)'));
      const lr = s().nodes.map((n) => n.position);
      expect(lr[1].x).toBeGreaterThan(lr[0].x);
      act(() => vi.advanceTimersByTime(100));
    } finally {
      vi.useRealTimers();
    }
  });

  it('disables auto layout without nodes and renders the dark theme', async () => {
    useUIStore.setState({ theme: 'dark' });
    const { wrapper } = await renderCanvas(false);
    expect(screen.getByTitle('Auto Layout (Top to Bottom)')).toBeDisabled();
    expect(wrapper().style.background).toBe('rgb(15, 23, 42)');
  });

  const dragFromHandle = (handle: HTMLElement, releaseOn: HTMLElement, point = { clientX: 300, clientY: 400 }) => {
    // reactflow finds the pressed handle with elementFromPoint, which jsdom cannot compute.
    vi.spyOn(document, 'elementFromPoint').mockReturnValueOnce(handle);
    fireEvent.mouseDown(handle, { button: 0, clientX: 10, clientY: 10 });
    fireEvent.mouseMove(document, point);
    fireEvent.mouseUp(releaseOn, point);
  };

  it('offers quick add when a connection is dropped on the pane and connects the new node', async () => {
    const { q, pane } = await renderCanvas();
    dragFromHandle(q('[data-nodeid="b"].source'), pane());
    expect(await screen.findByPlaceholderText('Search nodes...')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /AI Generator/ }));

    expect(screen.queryByPlaceholderText('Search nodes...')).not.toBeInTheDocument();
    const added = s().nodes.find((n) => n.data.type === NodeType.LLM)!;
    expect(added.position).toEqual({ x: 300, y: 400 });
    expect(s().edges.some((e) => e.source === 'b' && e.target === added.id)).toBe(true);
  });

  it('connects quick-added nodes upstream when dragging from a target handle', async () => {
    const { q, pane } = await renderCanvas();
    dragFromHandle(q('[data-nodeid="a"].target'), pane());
    await screen.findByPlaceholderText('Search nodes...');
    await userEvent.keyboard('{Enter}');
    const added = s().nodes.find((n) => n.data.type === NodeType.HTTP && !['a', 'b'].includes(n.id))!;
    expect(s().edges.some((e) => e.source === added.id && e.target === 'a')).toBe(true);
  });

  it('closes quick add with Escape and ignores connections released elsewhere', async () => {
    const { q, pane } = await renderCanvas();
    dragFromHandle(q('[data-nodeid="a"].source'), q('[data-testid="rf__node-b"]'));
    expect(screen.queryByPlaceholderText('Search nodes...')).not.toBeInTheDocument();

    dragFromHandle(q('[data-nodeid="a"].source'), pane());
    await screen.findByPlaceholderText('Search nodes...');
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByPlaceholderText('Search nodes...')).not.toBeInTheDocument());
  });
});
