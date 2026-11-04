import { describe, expect, it } from 'vitest';
import { MarkerType } from 'reactflow';
import { TEMPLATES, createCategories } from './index';
import { edge } from './types';
import { NodeStatus, NodeType, isValidNodeType } from '@/types';
import { renderHook } from '@testing-library/react';
import { useTranslation } from '@/store/translations';
import { useUIStore } from '@/store/uiStore';

const LANGUAGES = ['en', 'ru'] as const;
const dictionaryFor = (language: (typeof LANGUAGES)[number]) => {
  useUIStore.setState({ language });
  return renderHook(() => useTranslation()).result.current;
};

describe('edge helper', () => {
  it('builds a smoothstep edge with an arrow, adding the label only when given', () => {
    expect(edge('e1', 'a', 'b')).toEqual({
      id: 'e1',
      source: 'a',
      target: 'b',
      type: 'smoothstep',
      animated: false,
      markerEnd: { type: MarkerType.ArrowClosed },
    });
    expect(edge('e2', 'a', 'b', 'true')).toMatchObject({ label: 'true', labelStyle: { fontSize: 10 } });
  });
});

describe('createCategories', () => {
  it('builds the category tabs from the translations of every language', () => {
    for (const t of LANGUAGES.map(dictionaryFor)) {
      const categories = createCategories(t);
      expect(categories.map((c) => c.id)).toEqual(['all', 'basic', 'telegram', 'ai', 'data']);
      categories.forEach((c) => expect(typeof c.label === 'string' && c.label.length > 0).toBe(true));
    }
  });
});

describe('TEMPLATES', () => {
  const categoryIds = new Set(createCategories(dictionaryFor('en')).map((c) => c.id));

  it('have unique ids and known categories, and every category has a template', () => {
    const ids = TEMPLATES.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    TEMPLATES.forEach((t) => {
      expect(categoryIds.has(t.category)).toBe(true);
      expect(t.name).not.toBe('');
      expect(t.description).not.toBe('');
      expect(t.icon).toBeTruthy();
    });
    [...categoryIds].filter((id) => id !== 'all').forEach((id) => {
      expect(TEMPLATES.some((t) => t.category === id)).toBe(true);
    });
  });

  it.each(TEMPLATES.map((t) => [t.id, t] as const))('%s: nodes use existing node types and are idle custom nodes', (_, template) => {
    const nodeIds = template.nodes.map((n) => n.id);
    expect(new Set(nodeIds).size).toBe(nodeIds.length);
    template.nodes.forEach((node) => {
      expect(node.type).toBe('custom');
      expect(isValidNodeType(node.data.type)).toBe(true);
      expect(node.data.status).toBe(NodeStatus.IDLE);
      expect(node.data.label).not.toBe('');
      expect(node.data.config).toBeTypeOf('object');
      expect(Number.isFinite(node.position.x) && Number.isFinite(node.position.y)).toBe(true);
    });
  });

  it.each(TEMPLATES.map((t) => [t.id, t] as const))('%s: edges connect existing nodes into an acyclic graph', (_, template) => {
    const nodeIds = new Set(template.nodes.map((n) => n.id));
    const edgeIds = template.edges.map((e) => e.id);
    expect(new Set(edgeIds).size).toBe(edgeIds.length);

    const outgoing = new Map<string, string[]>();
    template.edges.forEach((e) => {
      expect(nodeIds.has(e.source)).toBe(true);
      expect(nodeIds.has(e.target)).toBe(true);
      expect(e.source).not.toBe(e.target);
      outgoing.set(e.source, [...(outgoing.get(e.source) ?? []), e.target]);
    });

    // Every node is connected.
    const connected = new Set(template.edges.flatMap((e) => [e.source, e.target]));
    expect([...nodeIds].filter((id) => !connected.has(id))).toEqual([]);

    // Kahn's algorithm: all nodes can be ordered, so there is no cycle.
    const indegree = new Map([...nodeIds].map((id) => [id, 0]));
    template.edges.forEach((e) => indegree.set(e.target, indegree.get(e.target)! + 1));
    const queue = [...nodeIds].filter((id) => indegree.get(id) === 0);
    let visited = 0;
    while (queue.length) {
      const id = queue.shift()!;
      visited++;
      (outgoing.get(id) ?? []).forEach((next) => {
        indegree.set(next, indegree.get(next)! - 1);
        if (indegree.get(next) === 0) queue.push(next);
      });
    }
    expect(visited).toBe(nodeIds.size);
  });

  it('labels the branches of conditional nodes "true" and "false"', () => {
    TEMPLATES.forEach((template) => {
      const conditionals = new Set(template.nodes.filter((n) => n.data.type === NodeType.CONDITIONAL).map((n) => n.id));
      template.edges
        .filter((e) => conditionals.has(e.source))
        .forEach((e) => expect(['true', 'false']).toContain(e.label));
      conditionals.forEach((id) => {
        expect(typeof template.nodes.find((n) => n.id === id)!.data.config.condition).toBe('string');
      });
    });
  });

  it('configure transform nodes with the keys the backend transform executor reads', () => {
    // go/pkg/executor/builtin/transform.go reads "type" and, per type, "template", "expression" or "filter".
    const transforms = TEMPLATES.flatMap((t) => t.nodes).filter((n) => n.data.type === NodeType.TRANSFORM);
    expect(transforms.length).toBeGreaterThan(0);
    transforms.forEach(({ data: { config } }) => {
      expect(['passthrough', 'template', 'expression', 'jq']).toContain(config.type);
      expect(config).not.toHaveProperty('language');
      const field = { template: 'template', expression: 'expression', jq: 'filter' }[config.type as string];
      if (field) expect(typeof config[field]).toBe('string');
    });
  });
});
