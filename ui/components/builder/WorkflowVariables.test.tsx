import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { WorkflowVariablesGuide } from './WorkflowVariablesGuide';
import { WorkflowVariablesEditor } from './WorkflowVariablesEditor';
import { useDagStore } from '@/store/dagStore';
import { resetBuilderStores, stubClipboard } from '@/test/builderFlowMocks';

beforeEach(resetBuilderStores);

describe('WorkflowVariablesGuide', () => {
  it('toggles categories and transform modes and copies examples', async () => {
    const user = userEvent.setup();
    const writeText = stubClipboard();
    render(<WorkflowVariablesGuide />);
    expect(screen.getByText('{{env.API_KEY}}')).toBeInTheDocument();
    expect(screen.queryByText('{{http.body}}')).not.toBeInTheDocument();
    expect(screen.queryByTitle('Close')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Node Outputs/ }));
    expect(screen.getByText('{{http.body}}')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Environment Variables/ }));
    expect(screen.queryByText('{{env.API_KEY}}')).not.toBeInTheDocument();

    expect(screen.queryByText('Passthrough')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Transform Modes/ }));
    expect(screen.getByText('Passthrough')).toBeInTheDocument();
    expect(screen.getByText('JQ')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Transform Modes/ }));
    expect(screen.queryByText('Passthrough')).not.toBeInTheDocument();

    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const copy = screen.getAllByTitle('Copy to clipboard')[0];
      fireEvent.click(copy);
      expect(writeText).toHaveBeenCalledWith('{{input.message}}');
      expect(copy.querySelector('.text-green-500')).not.toBeNull();
      act(() => vi.advanceTimersByTime(2100));
      expect(copy.querySelector('.text-green-500')).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it('renders as a modal closable by the button or the backdrop only', async () => {
    const onClose = vi.fn();
    const { container } = render(<WorkflowVariablesGuide isModal onClose={onClose} />);
    await userEvent.click(screen.getByText('Template Syntax'));
    expect(onClose).not.toHaveBeenCalled();
    await userEvent.click(container.firstChild as HTMLElement);
    expect(onClose).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByTitle('Close'));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it('ignores backdrop clicks without onClose', async () => {
    const { container } = render(<WorkflowVariablesGuide isModal />);
    await userEvent.click(container.firstChild as HTMLElement);
    expect(screen.queryByTitle('Close')).not.toBeInTheDocument();
  });
});

describe('WorkflowVariablesEditor', () => {
  const vars = () => useDagStore.getState().workflowVariables;
  const row = (key: string) => screen.getByText(key).parentElement as HTMLElement;

  it('renders nothing when closed', () => {
    const { container } = render(<WorkflowVariablesEditor isOpen={false} onClose={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('adds variables with normalised names via button and Enter', async () => {
    const onClose = vi.fn();
    render(<WorkflowVariablesEditor isOpen onClose={onClose} />);
    expect(screen.getByText('No variables defined yet')).toBeInTheDocument();
    const add = screen.getByRole('button', { name: 'Add' });
    expect(add).toBeDisabled();

    await userEvent.type(screen.getByPlaceholderText('VARIABLE_NAME'), 'base-url');
    expect(screen.getByPlaceholderText('VARIABLE_NAME')).toHaveValue('BASE_URL');
    await userEvent.type(screen.getByPlaceholderText('Value...'), 'http://x');
    await userEvent.click(add);
    expect(vars()).toEqual({ BASE_URL: 'http://x' });
    expect(screen.getByPlaceholderText('VARIABLE_NAME')).toHaveValue('');

    await userEvent.type(screen.getByPlaceholderText('VARIABLE_NAME'), 'empty');
    await userEvent.type(screen.getByPlaceholderText('Value...'), '{Enter}');
    expect(vars()).toEqual({ BASE_URL: 'http://x', EMPTY: '' });
    expect(screen.getByText('(empty)')).toBeInTheDocument();

    await userEvent.type(screen.getByPlaceholderText('Value...'), 'ignored{Enter}');
    expect(Object.keys(vars())).toHaveLength(2);

    await userEvent.click(screen.getAllByRole('button')[1]);
    expect(onClose).toHaveBeenCalled();
  });

  it('edits, cancels editing and deletes variables', async () => {
    useDagStore.setState({ workflowVariables: { REGION: 'eu', OTHER: 'x' } });
    render(<WorkflowVariablesEditor isOpen onClose={() => {}} />);

    await userEvent.click(within(row('REGION')).getByRole('button', { name: 'Edit' }));
    const input = screen.getByDisplayValue('eu');
    await userEvent.clear(input);
    await userEvent.type(input, 'us{Enter}');
    expect(vars().REGION).toBe('us');

    const group = () => screen.getByText('REGION').closest('.group') as HTMLElement;
    await userEvent.click(within(row('REGION')).getByRole('button', { name: 'Edit' }));
    await userEvent.clear(screen.getByDisplayValue('us'));
    await userEvent.type(within(group()).getByRole('textbox'), 'ap');
    await userEvent.click(within(group()).getAllByRole('button')[0]);
    expect(vars().REGION).toBe('ap');

    await userEvent.click(within(row('REGION')).getByRole('button', { name: 'Edit' }));
    await userEvent.type(screen.getByDisplayValue('ap'), 'zzz');
    await userEvent.click(within(group()).getAllByRole('button')[1]);
    expect(vars().REGION).toBe('ap');
    expect(screen.queryByDisplayValue('apzzz')).not.toBeInTheDocument();

    await userEvent.click(within(row('OTHER')).getByTitle('Delete'));
    expect(vars()).toEqual({ REGION: 'ap' });
  });

  it('masks secrets, reveals them and copies templates', async () => {
    const user = userEvent.setup();
    const writeText = stubClipboard();
    useDagStore.setState({ workflowVariables: { API_TOKEN: 'abcdef123', PIN_KEY: '12', PLAIN: 'v' } });
    render(<WorkflowVariablesEditor isOpen onClose={() => {}} />);
    expect(screen.getByText('ab••••••23')).toBeInTheDocument();
    expect(screen.getByText('••••••••')).toBeInTheDocument();
    expect(within(row('PLAIN')).queryByTitle('Show value')).not.toBeInTheDocument();

    await user.click(within(row('API_TOKEN')).getByTitle('Show value'));
    expect(screen.getByText('abcdef123')).toBeInTheDocument();
    await user.click(within(row('API_TOKEN')).getByTitle('Hide value'));
    expect(screen.getByText('ab••••••23')).toBeInTheDocument();

    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const copy = within(row('PLAIN')).getByTitle('Copy template');
      fireEvent.click(copy);
      expect(writeText).toHaveBeenCalledWith('{{env.PLAIN}}');
      expect(copy.querySelector('.text-green-500')).not.toBeNull();
      act(() => vi.advanceTimersByTime(2100));
      expect(copy.querySelector('.text-green-500')).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it('toggles the variables guide', async () => {
    render(<WorkflowVariablesEditor isOpen onClose={() => {}} />);
    await userEvent.click(screen.getByRole('button', { name: 'Variables Guide' }));
    expect(screen.getByText('Workflow Variables & Templates')).toBeInTheDocument();
    await userEvent.click(screen.getByTitle('Close'));
    expect(screen.queryByText('Workflow Variables & Templates')).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Variables Guide' }));
    await userEvent.click(screen.getByRole('button', { name: 'Variables Guide' }));
    expect(screen.queryByText('Workflow Variables & Templates')).not.toBeInTheDocument();
  });
});
