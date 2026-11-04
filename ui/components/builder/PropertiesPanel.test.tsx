import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PropertiesPanel } from './PropertiesPanel';
import { EdgeConfigPanel } from './EdgeConfigPanel';
import { useDagStore } from '@/store/dagStore';
import { NodeType } from '@/types';
import { makeEdge, makeNode, resetBuilderStores } from '@/test/builderFlowMocks';

const s = () => useDagStore.getState();
const node = (id: string) => s().nodes.find((n) => n.id === id)!;

beforeEach(resetBuilderStores);

describe('PropertiesPanel', () => {
  it('is hidden without a selected node', () => {
    const { container } = render(<PropertiesPanel />);
    expect((container.firstChild as HTMLElement).className).toContain('translate-x-full');
    expect(screen.queryByText('Properties')).not.toBeInTheDocument();
  });

  it('edits label and description of the selected node', async () => {
    useDagStore.setState({ nodes: [makeNode('n1', NodeType.COMMENT, { description: 'old' })], selectedNodeId: 'n1' });
    render(<PropertiesPanel />);
    expect(screen.getByText('ID: n1')).toBeInTheDocument();
    expect(screen.getByText('No specific configuration available for this node type.')).toBeInTheDocument();

    const name = screen.getByDisplayValue('Label n1');
    await userEvent.clear(name);
    await userEvent.type(name, 'Renamed');
    expect(node('n1').data.label).toBe('Renamed');
    expect(s().isDirty).toBe(true);

    const desc = screen.getByPlaceholderText('Description...');
    await userEvent.clear(desc);
    await userEvent.type(desc, 'New desc');
    expect(node('n1').data.description).toBe('New desc');
  });

  it('shows empty fields for nodes without label/description/config', () => {
    useDagStore.setState({
      nodes: [{ id: 'bare', type: 'custom', position: { x: 0, y: 0 }, data: { type: 'mystery' } } as never],
      selectedNodeId: 'bare',
    });
    render(<PropertiesPanel />);
    expect(screen.getAllByRole('textbox').map((e) => (e as HTMLInputElement).value)).toEqual(['', '']);
  });

  it('merges type defaults into the config editor and stores changes', async () => {
    useDagStore.setState({ nodes: [makeNode('d', NodeType.DELAY, { config: { duration: 5 } })], selectedNodeId: 'd' });
    render(<PropertiesPanel />);
    expect(screen.getByRole('spinbutton')).toHaveValue(5);
    await userEvent.selectOptions(screen.getAllByRole('combobox')[0], 'minutes');
    expect(node('d').data.config).toMatchObject({ duration: 5, unit: 'minutes' });
  });

  it('closes and deletes the node after confirmation', async () => {
    useDagStore.setState({ nodes: [makeNode('a'), makeNode('b')], selectedNodeId: 'a' });
    const { container } = render(<PropertiesPanel />);
    await userEvent.click(within(container.querySelector('.border-b') as HTMLElement).getByRole('button'));
    expect(s().selectedNodeId).toBeNull();

    useDagStore.setState({ selectedNodeId: 'b' });
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Node' }));
    expect(screen.getByText(/Are you sure you want to delete this node/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(s().nodes).toHaveLength(2);

    await userEvent.click(screen.getByRole('button', { name: 'Delete Node' }));
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));
    expect(s().nodes.map((n) => n.id)).toEqual(['a']);
    expect(screen.queryByText(/Are you sure/)).not.toBeInTheDocument();
  });
});

describe('EdgeConfigPanel', () => {
  const edge = () => s().edges[0];

  it('renders nothing without a selected edge', () => {
    const { container } = render(<EdgeConfigPanel />);
    expect(container).toBeEmptyDOMElement();
  });

  it('configures the branch and condition of an edge from a conditional node', async () => {
    useDagStore.setState({
      nodes: [makeNode('c', NodeType.CONDITIONAL), makeNode('t')],
      edges: [makeEdge('e1', 'c', 't', { data: { other: 1 } })],
      selectedEdgeId: 'e1',
    });
    render(<EdgeConfigPanel />);
    expect(screen.getByText('Label c')).toBeInTheDocument();
    expect(screen.getByText('Label t')).toBeInTheDocument();

    const trueBtn = screen.getByRole('button', { name: 'True Branch' });
    const falseBtn = screen.getByRole('button', { name: 'False Branch' });
    await userEvent.click(trueBtn);
    expect(trueBtn.className).toContain('border-green-500');
    await userEvent.click(falseBtn);
    expect(falseBtn.className).toContain('border-red-500');
    expect(trueBtn.className).not.toContain('border-green-500');

    await userEvent.type(screen.getByPlaceholderText('output.status == "success"'), 'output.ok');
    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }));
    expect(edge()).toMatchObject({ sourceHandle: 'false', data: { other: 1, condition: 'output.ok' } });
    expect(s().selectedEdgeId).toBeNull();
  });

  it('loads existing values, clears them on save and hides branches for regular nodes', async () => {
    useDagStore.setState({
      nodes: [],
      edges: [makeEdge('e1', 'x', 'y', { sourceHandle: 'true', data: { condition: 'a > 1' } })],
      selectedEdgeId: 'e1',
    });
    render(<EdgeConfigPanel />);
    expect(screen.getByText('x')).toBeInTheDocument();
    expect(screen.queryByText('True Branch')).not.toBeInTheDocument();
    const cond = screen.getByDisplayValue('a > 1');
    await userEvent.clear(cond);
    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }));
    expect(edge().data).toEqual({ condition: undefined });
    expect(edge().sourceHandle).toBe('true');
  });

  it('closes via cancel, backdrop and close button', async () => {
    useDagStore.setState({ edges: [makeEdge('e1', 'x', 'y')], selectedEdgeId: 'e1' });
    const { container } = render(<EdgeConfigPanel />);
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(s().selectedEdgeId).toBeNull();

    useDagStore.setState({ selectedEdgeId: 'e1' });
    await userEvent.click(await screen.findByText('Edge Configuration'));
    await userEvent.click(container.querySelector('.bg-black\\/20') as HTMLElement);
    expect(s().selectedEdgeId).toBeNull();

    useDagStore.setState({ selectedEdgeId: 'e1' });
    const header = (await screen.findByText('Edge Configuration')).closest('.border-b') as HTMLElement;
    await userEvent.click(within(header).getByRole('button'));
    expect(s().selectedEdgeId).toBeNull();
  });

  it('deletes the edge after confirmation', async () => {
    useDagStore.setState({ edges: [makeEdge('e1', 'x', 'y'), makeEdge('e2', 'y', 'z')], selectedEdgeId: 'e1' });
    render(<EdgeConfigPanel />);
    await userEvent.click(screen.getByRole('button', { name: 'Delete Edge' }));
    expect(screen.getByText(/Are you sure you want to delete this edge/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));
    expect(s().edges.map((e) => e.id)).toEqual(['e2']);
  });
});
