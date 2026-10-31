import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TriggerFormModal } from './TriggerFormModal';
import { triggerService } from '@/services/triggerService';
import { useUIStore } from '@/store/uiStore';
import { apiError } from '@/test/apiMock';
import type { Trigger } from '@/types/triggers';

const WF = '11111111-1111-4111-8111-111111111111';
const workflows = [
  { id: WF, name: 'Billing sync' },
  { id: '22222222-2222-4222-8222-222222222222', name: 'Reports' },
];

const existing = (overrides: Partial<Trigger> = {}): Trigger => ({
  id: 'trig-1',
  workflow_id: WF,
  name: 'Nightly',
  description: 'runs at night',
  type: 'cron',
  config: { schedule: '0 2 * * *', timezone: 'UTC', name: 'Nightly', keep_me: 1 },
  enabled: true,
  created_at: '',
  updated_at: '',
  ...overrides,
});

const setup = (props: Partial<React.ComponentProps<typeof TriggerFormModal>> = {}) => {
  const onClose = vi.fn();
  const onSaved = vi.fn();
  const user = userEvent.setup();
  const utils = render(
    <TriggerFormModal isOpen workflows={workflows} onClose={onClose} onSaved={onSaved} {...props} />,
  );
  return { ...utils, user, onClose, onSaved };
};

describe('TriggerFormModal', () => {
  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
  });

  it('renders nothing when closed', () => {
    render(<TriggerFormModal isOpen={false} workflows={[]} onClose={() => {}} onSaved={() => {}} />);
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('shows validation errors and does not submit an incomplete form', async () => {
    const create = vi.spyOn(triggerService, 'createTrigger');
    const { user } = setup();

    expect(screen.getByRole('heading', { name: 'Create New Trigger' })).toBeInTheDocument();
    await user.clear(screen.getByLabelText(/Cron expression/));
    await user.type(screen.getByLabelText(/Cron expression/), 'every day');
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));

    expect(create).not.toHaveBeenCalled();
    const alerts = screen.getAllByRole('alert').map((a) => a.textContent);
    expect(alerts).toEqual(['This field is required', 'This field is required', 'Not a valid cron expression']);

    // Editing a field clears its error.
    await user.type(screen.getByLabelText(/^Name/), 'x');
    expect(screen.getAllByRole('alert')).toHaveLength(2);
  });

  it('creates a cron trigger with the API payload', async () => {
    const saved = existing({ id: 'new' });
    const create = vi.spyOn(triggerService, 'createTrigger').mockResolvedValue(saved);
    const { user, onSaved } = setup();

    await user.selectOptions(screen.getByLabelText(/Workflow/), WF);
    await user.type(screen.getByLabelText(/^Name/), 'Hourly report');
    await user.type(screen.getByLabelText('Description'), 'Sends the report');
    await user.clear(screen.getByLabelText(/Cron expression/));
    await user.type(screen.getByLabelText(/Cron expression/), '0 9 * * 1-5');
    await user.type(screen.getByLabelText('Timezone'), 'Europe/Berlin');
    await user.click(screen.getByLabelText(/Enabled/));
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(saved, 'create'));
    expect(create).toHaveBeenCalledWith({
      workflow_id: WF,
      name: 'Hourly report',
      description: 'Sends the report',
      type: 'cron',
      config: { schedule: '0 9 * * 1-5', timezone: 'Europe/Berlin' },
      enabled: false,
    });
  });

  it.each([
    [
      'interval',
      async (user: ReturnType<typeof userEvent.setup>) => {
        await user.clear(screen.getByLabelText(/^Interval/));
        await user.type(screen.getByLabelText(/^Interval/), '90');
      },
      { interval: 90 },
    ],
    [
      'webhook',
      async (user: ReturnType<typeof userEvent.setup>) => {
        await user.type(screen.getByLabelText('Signing secret'), 's3cret');
        await user.type(screen.getByLabelText('Allowed IPs'), '10.0.0.0/8, 1.2.3.4');
      },
      { secret: 's3cret', ip_whitelist: ['10.0.0.0/8', '1.2.3.4'] },
    ],
    [
      'event',
      async (user: ReturnType<typeof userEvent.setup>) => {
        await user.type(screen.getByLabelText(/^Event type/), 'user.created');
        await user.type(screen.getByLabelText('Source'), 'auth');
        await user.type(screen.getByLabelText('Filter (JSON)'), '{{"plan": "pro"}');
      },
      { event_type: 'user.created', source: 'auth', filter: { plan: 'pro' } },
    ],
    ['manual', async () => {}, {}],
  ])('creates a %s trigger', async (type, fill, config) => {
    const create = vi.spyOn(triggerService, 'createTrigger').mockResolvedValue(existing());
    const { user } = setup({ defaultWorkflowId: WF });

    await user.type(screen.getByLabelText(/^Name/), 'T');
    await user.selectOptions(screen.getByLabelText('Trigger type'), type);
    expect(screen.getByTestId(`trigger-config-${type}`)).toBeInTheDocument();
    await fill(user);
    await user.type(screen.getByLabelText('Default input (JSON)'), '{{"source": "ui"}');
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));

    await waitFor(() => expect(create).toHaveBeenCalled());
    expect(create.mock.calls[0][0]).toMatchObject({ type, workflow_id: WF, name: 'T', enabled: true });
    expect(create.mock.calls[0][0].config).toEqual({ ...config, input: { source: 'ui' } });
  });

  it('validates per-type fields', async () => {
    const { user } = setup({ defaultWorkflowId: WF });
    await user.type(screen.getByLabelText(/^Name/), 'T');

    await user.selectOptions(screen.getByLabelText('Trigger type'), 'interval');
    await user.clear(screen.getByLabelText(/^Interval/));
    await user.type(screen.getByLabelText(/^Interval/), 'soon');
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));
    expect(screen.getByRole('alert')).toHaveTextContent('Not a valid positive duration');

    await user.selectOptions(screen.getByLabelText('Trigger type'), 'webhook');
    await user.type(screen.getByLabelText('Allowed IPs'), 'nope');
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));
    expect(screen.getAllByRole('alert').map((a) => a.textContent)).toContain('Contains an invalid IP address or CIDR range');

    await user.selectOptions(screen.getByLabelText('Trigger type'), 'event');
    await user.type(screen.getByLabelText('Filter (JSON)'), '[[1]');
    await user.type(screen.getByLabelText('Default input (JSON)'), 'nope');
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));
    const texts = screen.getAllByRole('alert').map((a) => a.textContent);
    expect(texts).toEqual(expect.arrayContaining(['This field is required', 'Must be a JSON object', 'Not valid JSON']));
  });

  it('shows server errors and keeps the dialog open', async () => {
    vi.spyOn(triggerService, 'createTrigger').mockRejectedValue(apiError(400, { message: 'invalid cron expression' }));
    const { user, onSaved } = setup({ defaultWorkflowId: WF });
    await user.type(screen.getByLabelText(/^Name/), 'T');
    await user.click(screen.getByRole('button', { name: 'Create trigger' }));

    expect(await screen.findByText('invalid cron expression')).toBeInTheDocument();
    expect(onSaved).not.toHaveBeenCalled();
  });

  it('edits an existing trigger, preserving unknown config keys', async () => {
    const update = vi.spyOn(triggerService, 'updateTrigger').mockResolvedValue(existing({ enabled: false }));
    const { user, onSaved } = setup({ trigger: existing() });

    expect(screen.getByRole('heading', { name: 'Edit Trigger' })).toBeInTheDocument();
    expect(screen.getByLabelText(/Workflow/)).toBeDisabled();
    expect(screen.getByLabelText(/^Name/)).toHaveValue('Nightly');
    expect(screen.getByLabelText(/Cron expression/)).toHaveValue('0 2 * * *');

    await user.clear(screen.getByLabelText(/Cron expression/));
    await user.type(screen.getByLabelText(/Cron expression/), '@daily');
    await user.click(screen.getByRole('button', { name: 'Save changes' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(expect.objectContaining({ id: 'trig-1' }), 'edit'));
    expect(update).toHaveBeenCalledWith('trig-1', {
      name: 'Nightly',
      description: 'runs at night',
      type: 'cron',
      config: { keep_me: 1, schedule: '@daily', timezone: 'UTC' },
      enabled: true,
    });
  });

  it('shows the webhook URL of an existing webhook trigger', async () => {
    const { user } = setup({ trigger: existing({ type: 'webhook', config: {} }) });

    const url = `${window.location.origin}/api/v1/webhooks/trig-1`;
    expect(screen.getByTestId('webhook-url')).toHaveTextContent(url);
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Copy' }));
    expect(await navigator.clipboard.readText()).toBe(url);
  });

  it('explains that the webhook URL appears after creation', async () => {
    const { user } = setup();
    await user.selectOptions(screen.getByLabelText('Trigger type'), 'webhook');
    expect(screen.getByText('The URL is available after the trigger is created.')).toBeInTheDocument();
  });

  it('lists workflows, a loading placeholder and unknown workflow ids', () => {
    const { rerender } = setup({ workflowsLoading: true, workflows: [] });
    expect(screen.getByRole('option', { name: 'Loading workflows...' })).toBeInTheDocument();
    rerender(
      <TriggerFormModal isOpen workflows={workflows} trigger={existing({ workflow_id: 'gone' })} onClose={() => {}} onSaved={() => {}} />,
    );
    expect(screen.getByRole('option', { name: 'Billing sync' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'gone' })).toBeInTheDocument();
  });

  it('cancels', async () => {
    const { user, onClose } = setup();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalled();
  });
});
