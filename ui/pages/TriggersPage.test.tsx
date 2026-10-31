import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TriggersPage } from './TriggersPage';
import { triggerService } from '@/services/triggerService';
import { workflowService } from '@/services/workflowService';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { apiError } from '@/test/apiMock';
import type { Trigger } from '@/types/triggers';

const WF = '11111111-1111-4111-8111-111111111111';

const trigger = (id: string, overrides: Partial<Trigger> = {}): Trigger => ({
  id,
  workflow_id: WF,
  name: `Trigger ${id}`,
  type: 'cron',
  config: { schedule: '0 * * * *' },
  enabled: true,
  created_at: '',
  updated_at: '',
  ...overrides,
});

const minutesFromNow = (m: number) => new Date(Date.now() + m * 60_000).toISOString();

const TRIGGERS: Trigger[] = [
  trigger('1', { description: 'hourly', last_run: minutesFromNow(-5), next_run: minutesFromNow(30) }),
  trigger('2', { type: 'interval', config: { interval: '5m' }, enabled: false, last_run: minutesFromNow(-180) }),
  trigger('3', { type: 'webhook', config: { secret: 'x' }, name: '', workflow_id: 'unknown-wf', last_run: minutesFromNow(-3 * 24 * 60) }),
  trigger('4', { type: 'event', config: { event_type: 'user.created' }, next_run: minutesFromNow(5 * 60) }),
  trigger('5', { type: 'manual', config: {}, next_run: minutesFromNow(3 * 24 * 60), last_run: minutesFromNow(0) }),
  trigger('6', { type: 'legacy' as never, config: {} }),
];

const rows = () => screen.getAllByRole('row').slice(1);
const rowFor = (name: string) => rows().find((r) => within(r).queryByText(name))!;
const toasts = () => useToastStore.getState().toasts;

describe('TriggersPage', () => {
  let getAll: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useToastStore.getState().clearAll();
    getAll = vi.spyOn(triggerService, 'getAllTriggers').mockResolvedValue(TRIGGERS);
    vi.spyOn(workflowService, 'getAll').mockResolvedValue([{ id: WF, name: 'Billing sync' }] as never);
  });

  it('renders triggers using the API shape (type, enabled, config, last_run)', async () => {
    render(<TriggersPage />);
    expect(await screen.findByText('Trigger 1')).toBeInTheDocument();

    const first = rowFor('Trigger 1');
    expect(within(first).getByText('hourly')).toBeInTheDocument();
    expect(within(first).getByText('Cron schedule')).toBeInTheDocument();
    expect(await within(first).findByText('Billing sync')).toBeInTheDocument();
    expect(within(first).getByText('0 * * * *')).toBeInTheDocument();
    expect(within(first).getByText(/Next run: in (29|30) mins/)).toBeInTheDocument();
    expect(within(first).getByRole('button', { name: /Enabled/ })).toHaveAttribute('aria-pressed', 'true');
    expect(within(first).getByText('5 mins ago')).toBeInTheDocument();

    const second = rowFor('Trigger 2');
    expect(within(second).getByText('every 5m')).toBeInTheDocument();
    expect(within(second).getByRole('button', { name: /Disabled/ })).toHaveAttribute('aria-pressed', 'false');
    expect(within(second).getByRole('button', { name: 'Run now' })).toBeDisabled();
    expect(within(second).getByText('3 hours ago')).toBeInTheDocument();

    const third = rowFor('webhook trigger 3');
    expect(within(third).getByText('unknown-wf')).toBeInTheDocument();
    expect(within(third).getByText('POST, HMAC-signed')).toBeInTheDocument();
    expect(within(third).getByText('3 days ago')).toBeInTheDocument();

    expect(within(rowFor('Trigger 4')).getByText(/Next run: in [45] hours/)).toBeInTheDocument();
    expect(within(rowFor('Trigger 5')).getByText(/Next run: in [23] days/)).toBeInTheDocument();
    expect(within(rowFor('Trigger 5')).getByText('just now')).toBeInTheDocument();
    expect(within(rowFor('Trigger 6')).getByText('legacy')).toBeInTheDocument();
    expect(within(rowFor('Trigger 6')).getByText('Never')).toBeInTheDocument();
  });

  it('filters by type on the server and by status locally', async () => {
    const user = userEvent.setup();
    render(<TriggersPage />);
    await screen.findByText('Trigger 1');
    expect(getAll).toHaveBeenLastCalledWith({ type: undefined });

    const [typeSelect, statusSelect] = screen.getAllByRole('combobox');
    await user.selectOptions(typeSelect, 'interval');
    await waitFor(() => expect(getAll).toHaveBeenLastCalledWith({ type: 'interval' }));

    await user.selectOptions(typeSelect, 'all');
    await user.selectOptions(statusSelect, 'disabled');
    await waitFor(() => expect(rows()).toHaveLength(1));
    expect(screen.getByText('Trigger 2')).toBeInTheDocument();

    getAll.mockResolvedValue([]);
    await user.selectOptions(statusSelect, 'enabled');
    expect(await screen.findByText('Try adjusting your filters')).toBeInTheDocument();
    await user.selectOptions(statusSelect, 'all');
    expect(await screen.findByText('Create your first trigger to automate workflow execution')).toBeInTheDocument();
  });

  it('enables and disables triggers', async () => {
    const user = userEvent.setup();
    const disable = vi.spyOn(triggerService, 'disableTrigger').mockResolvedValue(trigger('1', { enabled: false }));
    const enable = vi.spyOn(triggerService, 'enableTrigger').mockResolvedValue(trigger('2'));
    render(<TriggersPage />);
    await screen.findByText('Trigger 1');

    await user.click(within(rowFor('Trigger 1')).getByRole('button', { name: /Enabled/ }));
    expect(disable).toHaveBeenCalledWith('1');
    await user.click(within(rowFor('Trigger 2')).getByRole('button', { name: /Disabled/ }));
    expect(enable).toHaveBeenCalledWith('2');
    await waitFor(() => expect(getAll.mock.calls.length).toBeGreaterThanOrEqual(3));

    enable.mockRejectedValue(apiError(500, { message: 'db down' }));
    await user.click(within(rowFor('Trigger 2')).getByRole('button', { name: /Disabled/ }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', message: 'db down' }));
  });

  it('runs a trigger by hand and reports the result', async () => {
    const user = userEvent.setup();
    const execute = vi
      .spyOn(triggerService, 'executeTrigger')
      .mockResolvedValueOnce({ id: 'exec-9', workflow_id: WF, status: 'pending' })
      .mockResolvedValueOnce({} as never)
      .mockRejectedValueOnce(apiError(400, { code: 'TRIGGER_DISABLED', message: 'trigger is disabled' }));
    render(<TriggersPage />);
    await screen.findByText('Trigger 1');
    const run = () => user.click(within(rowFor('Trigger 1')).getByRole('button', { name: 'Run now' }));

    await run();
    expect(execute).toHaveBeenCalledWith('1');
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'success', message: 'Execution exec-9 started' }));
    await run();
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'success', message: undefined }));
    await run();
    await waitFor(() =>
      expect(toasts().at(-1)).toMatchObject({ type: 'error', message: 'trigger is disabled' }),
    );
  });

  it('deletes a trigger after confirmation', async () => {
    const user = userEvent.setup();
    const remove = vi.spyOn(triggerService, 'deleteTrigger').mockResolvedValueOnce().mockRejectedValueOnce(apiError(404, { message: 'gone' }));
    render(<TriggersPage />);
    await screen.findByText('Trigger 1');

    await user.click(within(rowFor('Trigger 1')).getByRole('button', { name: 'Delete' }));
    expect(screen.getByText(/Are you sure you want to delete this trigger/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(remove).not.toHaveBeenCalled();

    await user.click(within(rowFor('Trigger 1')).getByRole('button', { name: 'Delete' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
    expect(remove).toHaveBeenCalledWith('1');
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'success', title: 'Trigger deleted' }));

    await user.click(within(rowFor('Trigger 1')).getByRole('button', { name: 'Delete' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', message: 'gone' }));
  });

  it('opens the create dialog, saves and refreshes the list', async () => {
    const user = userEvent.setup();
    const create = vi.spyOn(triggerService, 'createTrigger').mockResolvedValue(trigger('new'));
    render(<TriggersPage />);
    await screen.findByText('Trigger 1');
    await screen.findAllByText('Billing sync');

    await user.click(screen.getByRole('button', { name: 'Create Trigger' }));
    const dialog = screen.getByRole('dialog');
    await user.selectOptions(within(dialog).getByLabelText(/Workflow/), WF);
    await user.type(within(dialog).getByLabelText(/^Name/), 'From page');
    await user.selectOptions(within(dialog).getByLabelText('Trigger type'), 'manual');
    await user.click(within(dialog).getByRole('button', { name: 'Create trigger' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(create).toHaveBeenCalledWith(expect.objectContaining({ workflow_id: WF, name: 'From page', type: 'manual', enabled: true }));
    expect(toasts().at(-1)).toMatchObject({ title: 'Trigger created' });
    await waitFor(() => expect(getAll.mock.calls.length).toBeGreaterThanOrEqual(2));
  });

  it('opens the edit dialog with the trigger values and closes it', async () => {
    const user = userEvent.setup();
    const update = vi.spyOn(triggerService, 'updateTrigger').mockResolvedValue(trigger('1'));
    render(<TriggersPage />);
    await screen.findByText('Trigger 1');

    await user.click(within(rowFor('Trigger 1')).getAllByRole('button', { name: 'Edit' })[0]);
    expect(screen.getByRole('heading', { name: 'Edit Trigger' })).toBeInTheDocument();
    expect(screen.getByLabelText(/^Name/)).toHaveValue('Trigger 1');
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ title: 'Trigger updated' }));
    expect(update).toHaveBeenCalledWith('1', expect.objectContaining({ type: 'cron' }));

    await user.click(within(rowFor('Trigger 2')).getAllByRole('button', { name: 'Edit' })[0]);
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('still lists triggers when workflows cannot be loaded, and shows the empty-state action', async () => {
    vi.spyOn(workflowService, 'getAll').mockRejectedValue(new Error('x'));
    getAll.mockResolvedValue([]);
    const user = userEvent.setup();
    render(<TriggersPage />);
    expect(await screen.findByText('No triggers found')).toBeInTheDocument();
    const buttons = screen.getAllByRole('button', { name: 'Create Trigger' });
    await user.click(buttons[buttons.length - 1]);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('shows load errors', async () => {
    getAll.mockRejectedValue(apiError(500, { message: 'triggers unavailable' }));
    render(<TriggersPage />);
    expect(await screen.findByText('triggers unavailable')).toBeInTheDocument();
  });
});
