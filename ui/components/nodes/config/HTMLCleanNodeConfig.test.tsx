import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { HTMLCleanNodeConfigComponent } from './HTMLCleanNodeConfig';
import type { HTMLCleanNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('HTMLCleanNodeConfig', () => {
  const t = nodesEnT();
  const h = t.nodeConfig.htmlClean;

  it('defaults to both formats with metadata fields listed', () => {
    render(<HTMLCleanNodeConfigComponent config={undefined as unknown as HTMLCleanNodeConfig} onChange={() => {}} />);
    expect(nodesField<HTMLSelectElement>(h.outputFormat)).toHaveValue('both');
    expect(nodesField(h.extractMetadata)).toBeChecked();
    expect(nodesField(h.preserveLinks)).not.toBeChecked();
    expect(screen.getByText('text_content')).toBeInTheDocument();
    expect(screen.getByText('html_content')).toBeInTheDocument();
    expect(screen.getByText('excerpt')).toBeInTheDocument();
  });

  it('updates the output preview and emits changes', async () => {
    const { user, last } = renderConfig(HTMLCleanNodeConfigComponent, {
      output_format: 'both',
      extract_metadata: true,
      preserve_links: false,
    });
    await user.selectOptions(nodesField<HTMLSelectElement>(h.outputFormat), 'text');
    expect(last()).toEqual({ output_format: 'text', extract_metadata: true, preserve_links: false, max_length: 0, input_key: '' });
    expect(screen.queryByText('html_content')).toBeNull();

    await user.selectOptions(nodesField<HTMLSelectElement>(h.outputFormat), 'html');
    expect(screen.queryByText('text_content')).toBeNull();
    expect(screen.getByText('html_content')).toBeInTheDocument();

    await user.click(nodesField(h.extractMetadata));
    expect(last().extract_metadata).toBe(false);
    expect(screen.queryByText('excerpt')).toBeNull();

    await user.click(nodesField(h.preserveLinks));
    expect(last().preserve_links).toBe(true);

    const max = screen.getByPlaceholderText(h.maxLengthPlaceholder);
    fireEvent.change(max, { target: { value: '500' } });
    expect(last().max_length).toBe(500);
    fireEvent.change(max, { target: { value: '-1' } });
    expect(last().max_length).toBe(0);
    fireEvent.change(max, { target: { value: '' } });
    expect(last().max_length).toBe(0);

    await user.type(screen.getByPlaceholderText(h.inputKeyPlaceholder), 'html');
    expect(last().input_key).toBe('html');
  });
});
