import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { WorkflowCard } from './WorkflowCard';
import type { DAG } from '@/types';
import { makeNode } from '@/test/builderFlowMocks';

const translations = {
  edit: 'Edit',
  cloneTooltip: 'Clone',
  deleteTooltip: 'Delete',
  created: 'Created',
  updated: 'Updated',
  nodesCount: 'nodes',
  node: 'node',
};

const workflow = (overrides: Partial<DAG> = {}): DAG =>
  ({
    id: 'wf1',
    name: 'Billing',
    description: 'Sync invoices',
    status: 'active',
    nodes: [makeNode('a'), makeNode('b')],
    edges: [],
    createdAt: '2024-01-01',
    updatedAt: '2024-02-02',
    ...overrides,
  }) as DAG;

const renderCard = (wf: DAG) => {
  const handlers = { onEdit: vi.fn(), onClone: vi.fn(), onDelete: vi.fn() };
  render(<WorkflowCard workflow={wf} {...handlers} formatDate={(d) => `<${d}>`} translations={translations} />);
  return handlers;
};

describe('WorkflowCard', () => {
  it('renders workflow info and fires edit/clone/delete', async () => {
    const wf = workflow();
    const h = renderCard(wf);
    expect(screen.getByText('Billing')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getByText('Sync invoices')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('nodes')).toBeInTheDocument();
    expect(screen.getByText('Created <2024-01-01>')).toBeInTheDocument();
    expect(screen.getByText('Updated <2024-02-02>')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }));
    expect(h.onEdit).toHaveBeenCalledWith('wf1');
    await userEvent.click(screen.getByTitle('Clone'));
    expect(h.onClone).toHaveBeenCalledWith(wf);
    await userEvent.click(screen.getByTitle('Delete'));
    expect(h.onDelete).toHaveBeenCalledWith('wf1', 'Billing');
  });

  it.each(['draft', 'inactive', 'archived'] as const)('renders the %s status and singular node label', (status) => {
    renderCard(workflow({ status, description: '', nodes: [makeNode('x')] }));
    expect(screen.getByText(status[0].toUpperCase() + status.slice(1))).toBeInTheDocument();
    expect(screen.getByText('node')).toBeInTheDocument();
  });

  it('handles a workflow without nodes', () => {
    renderCard(workflow({ nodes: undefined as never }));
    expect(screen.getByText('0')).toBeInTheDocument();
    expect(screen.getByText('nodes')).toBeInTheDocument();
  });
});
