import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Zap } from 'lucide-react';
import { CategoryTabs, TemplateCard } from '@/components/templates';
import { useUIStore } from '@/store/uiStore';
import type { CategoryOption, WorkflowTemplate } from '@/data/workflowTemplates';
import { makeEdge, makeNode } from '@/test/builderFlowMocks';

const categories: CategoryOption[] = [
  { id: 'all', label: 'All Templates' },
  { id: 'ai', label: 'AI & LLM' },
];

const template: WorkflowTemplate = {
  id: 't1',
  name: 'Summariser',
  description: 'Summarise things',
  icon: Zap,
  color: 'purple',
  category: 'ai',
  nodes: [makeNode('a'), makeNode('b')],
  edges: [makeEdge('e', 'a', 'b')],
};

beforeEach(() => useUIStore.setState({ language: 'en' }));

describe('CategoryTabs', () => {
  it('renders every category, highlights the selected one and reports clicks', async () => {
    const onChange = vi.fn();
    render(<CategoryTabs categories={categories} selectedCategory="ai" onCategoryChange={onChange} />);
    expect(screen.getByRole('button', { name: 'AI & LLM' }).className).toContain('bg-blue-100');
    expect(screen.getByRole('button', { name: 'All Templates' }).className).not.toContain('bg-blue-100');
    await userEvent.click(screen.getByRole('button', { name: 'All Templates' }));
    expect(onChange).toHaveBeenCalledWith('all');
  });
});

describe('TemplateCard', () => {
  it('shows template details and category label and calls onClick with the template', async () => {
    const onClick = vi.fn();
    render(<TemplateCard template={template} categories={categories} onClick={onClick} />);
    expect(screen.getByText('Summariser')).toBeInTheDocument();
    expect(screen.getByText('Summarise things')).toBeInTheDocument();
    expect(screen.getByText('2 nodes')).toBeInTheDocument();
    expect(screen.getByText('1 edges')).toBeInTheDocument();
    expect(screen.getByText('AI & LLM')).toBeInTheDocument();
    expect(screen.getByText('Use')).toBeInTheDocument();
    await userEvent.click(screen.getByText('Summariser'));
    expect(onClick).toHaveBeenCalledWith(template);
  });

  it('renders without a category label when the category is unknown', () => {
    render(<TemplateCard template={{ ...template, category: 'data' }} categories={categories} onClick={() => {}} />);
    expect(screen.queryByText('AI & LLM')).not.toBeInTheDocument();
  });
});
