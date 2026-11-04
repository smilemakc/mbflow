import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TemplatesModal } from './TemplatesModal';
import { useUIStore } from '@/store/uiStore';
import { useDagStore } from '@/store/dagStore';
import { TEMPLATES } from '@/data/workflowTemplates';

const cardTitles = () => screen.queryAllByRole('heading', { level: 3 }).map((h) => h.textContent);

describe('TemplatesModal', () => {
  beforeEach(() => {
    useUIStore.setState({ language: 'en', activeModal: 'templates' });
    useDagStore.setState({ nodes: [], edges: [], isDirty: true });
  });

  it('lists all templates with node/edge counts', () => {
    render(<TemplatesModal />);
    expect(screen.getByRole('heading', { name: /Workflow Templates/ })).toBeInTheDocument();
    expect(cardTitles()).toEqual(TEMPLATES.map((t) => t.name));
    const first = screen.getByRole('heading', { name: TEMPLATES[0].name }).parentElement!;
    expect(within(first).getByText(`${TEMPLATES[0].nodes.length} nodes`)).toBeInTheDocument();
    expect(within(first).getByText(`${TEMPLATES[0].edges.length} edges`)).toBeInTheDocument();
  });

  it('filters by category and search text, with a no-results state', async () => {
    const user = userEvent.setup();
    render(<TemplatesModal />);

    await user.click(screen.getByRole('button', { name: 'Telegram' }));
    expect(cardTitles()).toEqual(TEMPLATES.filter((t) => t.category === 'telegram').map((t) => t.name));

    await user.click(screen.getByRole('button', { name: 'All Templates' }));
    await user.type(screen.getByPlaceholderText('Search templates...'), 'health');
    expect(cardTitles()).toEqual(['Scheduled Health Monitor']);

    await user.click(screen.getByRole('button', { name: 'AI & LLM' }));
    expect(cardTitles()).toEqual([]);
    expect(screen.getByText('No templates found matching your criteria')).toBeInTheDocument();
  });

  it('asks for confirmation and loads a deep copy of the template graph', async () => {
    const user = userEvent.setup();
    const template = TEMPLATES[0];
    render(<TemplatesModal />);

    await user.click(screen.getByRole('heading', { name: template.name }));
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText(/This will replace your current workflow/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Load Template' }));

    const { nodes, edges, isDirty } = useDagStore.getState();
    expect(nodes).toEqual(template.nodes);
    expect(edges).toEqual(template.edges);
    expect(nodes).not.toBe(template.nodes);
    expect(nodes[0]).not.toBe(template.nodes[0]);
    expect(isDirty).toBe(false);
    expect(useUIStore.getState().activeModal).toBeNull();
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('cancelling the confirmation keeps the current graph', async () => {
    const user = userEvent.setup();
    render(<TemplatesModal />);
    await user.click(screen.getByRole('heading', { name: TEMPLATES[1].name }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(useDagStore.getState().nodes).toEqual([]);
    expect(useUIStore.getState().activeModal).toBe('templates');
  });

  it('closes via the header button', async () => {
    render(<TemplatesModal />);
    await userEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(useUIStore.getState().activeModal).toBeNull();
  });
});
