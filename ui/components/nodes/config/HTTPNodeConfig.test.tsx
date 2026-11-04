import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { HTTPNodeConfigComponent } from './HTTPNodeConfig';
import type { HTTPNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

const base: HTTPNodeConfig = { url: 'https://api.test', method: 'GET', headers: {}, body: '' };

describe('HTTPNodeConfig', () => {
  const t = nodesEnT();
  const h = t.nodeConfig.http;

  it('normalises a config without headers/body and emits it', () => {
    const { last } = renderConfig(HTTPNodeConfigComponent, { url: '', method: 'GET' } as HTTPNodeConfig);
    expect(last()).toEqual({ url: '', method: 'GET', headers: {}, body: '' });
    expect(nodesField(h.timeout)).toHaveValue(30);
    expect(nodesField(h.retryCount)).toHaveValue(0);
    expect(nodesField(h.followRedirects)).toBeChecked();
  });

  it('shows the body only for POST/PUT/PATCH', async () => {
    const { user, last } = renderConfig(HTTPNodeConfigComponent, base);
    expect(screen.queryByPlaceholderText(h.bodyPlaceholder)).toBeNull();
    await user.selectOptions(nodesField<HTMLSelectElement>(h.method), 'POST');
    expect(last().method).toBe('POST');
    const body = screen.getByPlaceholderText(h.bodyPlaceholder);
    fireEvent.change(body, { target: { value: '{"a":1}' } });
    expect(last().body).toBe('{"a":1}');
    await user.selectOptions(nodesField<HTMLSelectElement>(h.method), 'DELETE');
    expect(screen.queryByPlaceholderText(h.bodyPlaceholder)).toBeNull();
  });

  it('edits url, timeout, retries and redirects', async () => {
    const { user, last } = renderConfig(HTTPNodeConfigComponent, base);
    await user.type(screen.getByPlaceholderText(h.urlPlaceholder), '/v1');
    expect(last().url).toBe('https://api.test/v1');
    fireEvent.change(nodesField(h.timeout), { target: { value: '60' } });
    expect(last().timeout_seconds).toBe(60);
    fireEvent.change(nodesField(h.retryCount), { target: { value: '3' } });
    expect(last().retry_count).toBe(3);
    await user.click(nodesField(h.followRedirects));
    expect(last().follow_redirects).toBe(false);
  });

  it('adds, renames, edits and removes headers', async () => {
    const { user, last } = renderConfig(HTTPNodeConfigComponent, { ...base, headers: { header1: 'x' } });
    await user.click(screen.getByRole('button', { name: h.addHeader }));
    expect(last().headers).toEqual({ header1: 'x', header2: '' });

    const keys = screen.getAllByPlaceholderText(h.headerKeyPlaceholder);
    fireEvent.change(keys[1], { target: { value: 'Authorization' } });
    expect(last().headers).toEqual({ header1: 'x', Authorization: '' });

    const values = screen.getAllByPlaceholderText(h.headerValuePlaceholder);
    fireEvent.change(values[1], { target: { value: 'Bearer t' } });
    expect(last().headers).toEqual({ header1: 'x', Authorization: 'Bearer t' });

    await user.click(screen.getAllByTitle(h.removeHeader)[0]);
    expect(last().headers).toEqual({ Authorization: 'Bearer t' });
  });

  it('resyncs from the parent', () => {
    const { setConfig } = renderConfig(HTTPNodeConfigComponent, base);
    act(() => setConfig({ url: 'https://other', method: 'PUT', headers: { A: 'b' }, body: 'x' }));
    expect(screen.getByPlaceholderText(h.urlPlaceholder)).toHaveValue('https://other');
    expect(screen.getByPlaceholderText(h.bodyPlaceholder)).toHaveValue('x');
    expect(screen.getByDisplayValue('A')).toBeInTheDocument();
  });
});
