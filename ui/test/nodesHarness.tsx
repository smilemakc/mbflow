/**
 * Test helpers for node config editors (components/nodes/**).
 *
 * renderConfig mounts a config component inside a stateful parent that feeds
 * every onChange payload back as the new `config` prop, like the node panel does.
 */
import React, { useState } from 'react';
import { vi } from 'vitest';
import { render, renderHook, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { useUIStore } from '@/store/uiStore';
import { useTranslation } from '@/store/translations';

type ConfigComponent<T> = React.ComponentType<{ config: T; nodeId?: string; onChange: (c: T) => void }>;

export function renderConfig<T>(
  Component: ConfigComponent<T>,
  initial: T,
  options: { controlled?: boolean; router?: boolean; nodeId?: string } = {},
) {
  const { controlled = true, router = false, nodeId = 'node-1' } = options;
  const onChange = vi.fn<(c: T) => void>();
  let setFromOutside: (c: T) => void = () => {};

  function Harness() {
    const [config, setConfig] = useState<T>(initial);
    setFromOutside = setConfig;
    return (
      <Component
        config={config}
        nodeId={nodeId}
        onChange={(c) => {
          onChange(c);
          if (controlled) setConfig(c);
        }}
      />
    );
  }

  const user = userEvent.setup();
  const utils = render(router ? <MemoryRouter><Harness /></MemoryRouter> : <Harness />);
  return {
    ...utils,
    user,
    onChange,
    /** Last payload passed to onChange. */
    last: (): T => onChange.mock.lastCall?.[0] as T,
    /** Replaces the config prop from the parent, e.g. another node selected. */
    setConfig: (c: T) => setFromOutside(c),
  };
}

/** English dictionary, with the UI language set to English. */
export function nodesEnT() {
  useUIStore.setState({ language: 'en' });
  return renderHook(() => useTranslation()).result.current;
}

const CONTROL = 'input, select, textarea';

const ownText = (el: Element) =>
  Array.from(el.childNodes)
    .filter((n) => n.nodeType === Node.TEXT_NODE)
    .map((n) => n.textContent)
    .join('')
    .trim();

/**
 * The form control belonging to a visible label text: the first input/select/textarea
 * in the nearest ancestor of the element whose own text starts with `text`.
 */
export function nodesField<E extends HTMLElement = HTMLInputElement>(text: string, index = 0): E {
  const labels = Array.from(document.body.querySelectorAll('label, span, h4, h5, p')).filter((el) =>
    ownText(el).startsWith(text),
  );
  const label = labels[index];
  if (!label) throw new Error(`No label "${text}" (found ${labels.length})`);
  let node: Element | null = label;
  while (node) {
    const control = node.matches(CONTROL) ? node : node.querySelector(CONTROL);
    if (control) return control as E;
    node = node.parentElement;
  }
  throw new Error(`No control for label "${text}"`);
}

export { screen };
