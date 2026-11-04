import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { RSSParserNodeConfigComponent } from './RSSParserNodeConfig';
import type { RSSParserNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('RSSParserNodeConfig', () => {
  const t = nodesEnT();
  const r = t.nodeConfig.rssParser;

  it('renders safe defaults for a missing config', () => {
    render(<RSSParserNodeConfigComponent config={undefined as unknown as RSSParserNodeConfig} onChange={() => {}} />);
    expect(screen.getByPlaceholderText(r.urlPlaceholder)).toHaveValue('');
    expect(screen.getByPlaceholderText(r.maxItemsPlaceholder)).toHaveValue(null);
    expect(nodesField(r.includeContent)).not.toBeChecked();
    expect(screen.queryByText('(full article)')).toBeNull();
  });

  it('emits url, max items and include-content changes', async () => {
    const { user, last } = renderConfig(RSSParserNodeConfigComponent, { url: '', maxItems: 0, includeContent: false });
    await user.type(screen.getByPlaceholderText(r.urlPlaceholder), 'https://x.io/feed');
    expect(last()).toEqual({ url: 'https://x.io/feed', maxItems: 0, includeContent: false });

    const max = screen.getByPlaceholderText(r.maxItemsPlaceholder);
    fireEvent.change(max, { target: { value: '25' } });
    expect(last().maxItems).toBe(25);
    fireEvent.change(max, { target: { value: '-4' } });
    expect(last().maxItems).toBe(0);
    fireEvent.change(max, { target: { value: 'abc' } });
    expect(last().maxItems).toBe(0);

    await user.click(nodesField(r.includeContent));
    expect(last().includeContent).toBe(true);
    expect(screen.getByText('(full article)')).toBeInTheDocument();
  });
});
