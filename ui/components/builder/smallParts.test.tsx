import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ContextMenu } from './ContextMenu';
import { QuickAddMenu } from './QuickAddMenu';
import { SubWorkflowProgress } from './SubWorkflowProgress';
import { ResourceSelector } from './ResourceSelector';
import { NODE_TEMPLATE_MIME, readNodeTemplate } from './dragData';
import { useDagStore } from '@/store/dagStore';
import { NodeType } from '@/types';
import { resetBuilderStores } from '@/test/builderFlowMocks';

beforeEach(resetBuilderStores);

const dataTransfer = (values: Record<string, string>) => ({ getData: (k: string) => values[k] ?? '' }) as DataTransfer;

describe('readNodeTemplate', () => {
  it('parses a template, ignores missing or invalid data', () => {
    expect(readNodeTemplate(dataTransfer({ [NODE_TEMPLATE_MIME]: '{"label":"X"}' }))).toEqual({ label: 'X' });
    expect(readNodeTemplate(dataTransfer({}))).toBeUndefined();
    expect(readNodeTemplate(dataTransfer({ [NODE_TEMPLATE_MIME]: '{bad' }))).toBeUndefined();
  });
});

describe('ContextMenu', () => {
  it('offers node actions, forwards them with the target id and closes', async () => {
    const onAction = vi.fn();
    const onClose = vi.fn();
    render(<ContextMenu x={5} y={6} type="node" targetId="n1" onAction={onAction} onClose={onClose} />);
    for (const [label, action] of [['Duplicate', 'duplicate'], ['Copy ID', 'copy_id'], ['Delete', 'delete']]) {
      await userEvent.click(screen.getByRole('button', { name: label }));
      expect(onAction).toHaveBeenLastCalledWith(action, 'n1');
    }
    expect(onClose).toHaveBeenCalledTimes(3);
    expect(screen.queryByText('Reset View')).not.toBeInTheDocument();
  });

  it('offers pane actions and closes on outside click only', async () => {
    const onAction = vi.fn();
    const onClose = vi.fn();
    render(<ContextMenu x={0} y={0} type="pane" onAction={onAction} onClose={onClose} />);
    fireEvent.mouseDown(screen.getByRole('button', { name: 'Reset View' }));
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.mouseDown(document.body);
    expect(onClose).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByRole('button', { name: 'Reset View' }));
    expect(onAction).toHaveBeenCalledWith('fit_view', undefined);
    expect(screen.queryByText('Duplicate')).not.toBeInTheDocument();
  });
});

describe('QuickAddMenu', () => {
  const setup = () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    render(<QuickAddMenu x={1} y={2} onSelect={onSelect} onClose={onClose} />);
    return { onSelect, onClose, input: screen.getByPlaceholderText('Search nodes...') };
  };

  it('focuses search, filters items and selects by click', async () => {
    const { onSelect, input } = setup();
    expect(input).toHaveFocus();
    expect(screen.getAllByRole('button')).toHaveLength(5);
    await userEvent.type(input, 'tele');
    expect(screen.getAllByRole('button')).toHaveLength(1);
    await userEvent.click(screen.getByRole('button', { name: /Telegram Bot/ }));
    expect(onSelect).toHaveBeenCalledWith(NodeType.TELEGRAM);
  });

  it('navigates with arrow keys and selects with Enter, closes on Escape', async () => {
    const { onSelect, onClose } = setup();
    await userEvent.keyboard('{ArrowUp}');
    await userEvent.keyboard('{Enter}');
    expect(onSelect).toHaveBeenLastCalledWith(NodeType.DELAY);
    await userEvent.keyboard('{ArrowDown}{ArrowDown}{Enter}');
    expect(onSelect).toHaveBeenLastCalledWith(NodeType.LLM);
    await userEvent.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalled();
  });

  it('shows an empty state, ignores Enter and arrows without results', async () => {
    const { onSelect, input } = setup();
    await userEvent.type(input, 'zzz');
    expect(screen.getByText('No nodes found')).toBeInTheDocument();
    await userEvent.keyboard('{ArrowDown}{ArrowUp}{Enter}');
    expect(onSelect).not.toHaveBeenCalled();
    await userEvent.clear(input);
    await userEvent.keyboard('{Enter}');
    expect(onSelect).toHaveBeenCalledWith(NodeType.HTTP);
  });

  it('closes on outside mousedown', () => {
    const { onClose, input } = setup();
    fireEvent.mouseDown(input);
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.mouseDown(document.body);
    expect(onClose).toHaveBeenCalled();
  });
});

describe('SubWorkflowProgress', () => {
  it('renders nothing for zero items', () => {
    const { container } = render(<SubWorkflowProgress total={0} completed={0} failed={0} running={0} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows counts and progress widths', () => {
    const { container } = render(<SubWorkflowProgress total={10} completed={5} failed={2} running={1} />);
    expect(screen.getByText('7/10')).toBeInTheDocument();
    expect(screen.getByText('done:5')).toBeInTheDocument();
    expect(screen.getByText('run:1')).toBeInTheDocument();
    expect(screen.getByText('fail:2')).toBeInTheDocument();
    expect(screen.getByText('wait:2')).toBeInTheDocument();
    expect((container.querySelector('.bg-green-500') as HTMLElement).style.width).toBe('50%');
  });

  it('hides zero counters', () => {
    render(<SubWorkflowProgress total={2} completed={2} failed={0} running={0} />);
    expect(screen.getByText('done:2')).toBeInTheDocument();
    expect(screen.queryByText(/run:|fail:|wait:/)).not.toBeInTheDocument();
  });
});

describe('ResourceSelector', () => {
  it('lists attached resources filtered by type and reports the template value', async () => {
    useDagStore.setState({
      resources: [
        { resource_id: 'r1', alias: 'files', access_type: 'read', resource_type: 'file_storage', resource_name: 'Main' },
        { resource_id: 'r2', alias: 'other', access_type: 'read', resource_type: 'credentials' },
      ] as never,
    });
    const onChange = vi.fn();
    render(<ResourceSelector value="" onChange={onChange} label="Storage" hint="Pick one" resourceType="file_storage" />);
    expect(screen.getByText('Storage')).toBeInTheDocument();
    expect(screen.getByText('Pick one')).toBeInTheDocument();
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Select a resource...', 'files (Main)']);
    await userEvent.selectOptions(screen.getByRole('combobox'), '{{resource.files}}');
    expect(onChange).toHaveBeenCalledWith('{{resource.files}}');
    expect(screen.queryByText(/No resources attached/)).not.toBeInTheDocument();
  });

  it('warns when nothing is attached', () => {
    render(<ResourceSelector value="" onChange={() => {}} placeholder="Choose" />);
    expect(screen.getByRole('option', { name: 'Choose' })).toBeInTheDocument();
    expect(screen.getByText(/No resources attached/)).toBeInTheDocument();
  });

  it('lists all resources without a type filter', () => {
    useDagStore.setState({ resources: [{ resource_id: 'r2', alias: 'other', access_type: 'read' }] as never });
    render(<ResourceSelector value="" onChange={() => {}} />);
    expect(screen.getByRole('option', { name: 'other' })).toBeInTheDocument();
  });
});
