import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { TelegramCallbackNodeConfigComponent } from './TelegramCallbackNodeConfig';
import type { TelegramCallbackNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('TelegramCallbackNodeConfig', () => {
  const t = nodesEnT();
  const c = t.nodeConfig.telegramCallback;

  it('renders defaults and emits field changes', async () => {
    const { user, last } = renderConfig(TelegramCallbackNodeConfigComponent, {} as TelegramCallbackNodeConfig);
    expect(nodesField(c.cacheTime)).toHaveValue(0);
    expect(nodesField(c.timeout)).toHaveValue(null);
    expect(screen.getByLabelText(c.showAlert)).not.toBeChecked();

    fireEvent.change(screen.getByPlaceholderText('{{env.TELEGRAM_BOT_TOKEN}}'), { target: { value: 'T' } });
    expect(last()).toMatchObject({ bot_token: 'T', callback_query_id: '', show_alert: false, cache_time: 0 });

    await user.type(screen.getByPlaceholderText(c.callbackQueryIdPlaceholder), 'q1');
    expect(last().callback_query_id).toBe('q1');

    await user.type(screen.getByPlaceholderText(c.textResponsePlaceholder), 'Done');
    expect(last().text).toBe('Done');

    await user.click(screen.getByLabelText(c.showAlert));
    expect(last().show_alert).toBe(true);

    await user.type(screen.getByPlaceholderText(c.urlPlaceholder), 'u');
    expect(last().url).toBe('u');
  });

  it('parses numeric fields, treating empty values as defaults', () => {
    const { last } = renderConfig(TelegramCallbackNodeConfigComponent, {
      bot_token: 'x',
      callback_query_id: 'y',
      cache_time: 10,
      timeout: 15,
    } as TelegramCallbackNodeConfig);
    fireEvent.change(nodesField(c.cacheTime), { target: { value: '300' } });
    expect(last().cache_time).toBe(300);
    fireEvent.change(nodesField(c.cacheTime), { target: { value: '' } });
    expect(last().cache_time).toBe(0);
    fireEvent.change(nodesField(c.timeout), { target: { value: '45' } });
    expect(last().timeout).toBe(45);
    fireEvent.change(nodesField(c.timeout), { target: { value: '' } });
    expect(last().timeout).toBeUndefined();
  });

  it('resyncs when the parent passes a new config', () => {
    const { setConfig } = renderConfig(TelegramCallbackNodeConfigComponent, { bot_token: 'a', callback_query_id: 'b' } as TelegramCallbackNodeConfig);
    act(() => setConfig({ bot_token: 'z', callback_query_id: 'q', show_alert: true, url: 'https://x', cache_time: 9 }));
    expect(screen.getByPlaceholderText('{{env.TELEGRAM_BOT_TOKEN}}')).toHaveValue('z');
    expect(screen.getByLabelText(c.showAlert)).toBeChecked();
    expect(screen.getByPlaceholderText(c.urlPlaceholder)).toHaveValue('https://x');
    expect(nodesField(c.cacheTime)).toHaveValue(9);
  });
});
