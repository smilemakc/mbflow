import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { TelegramDownloadNodeConfig } from './TelegramDownloadNodeConfig';
import type { TelegramDownloadNodeConfig as Config } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('TelegramDownloadNodeConfig', () => {
  const t = nodesEnT();
  const d = t.nodeConfig.telegramDownload;

  it('shows defaults for an empty config', () => {
    renderConfig(TelegramDownloadNodeConfig, {} as Config);
    expect(screen.getByPlaceholderText('{{env.TELEGRAM_BOT_TOKEN}}')).toHaveValue('');
    expect(screen.getByPlaceholderText(d.fileIdPlaceholder)).toHaveValue('');
    expect(screen.getByRole('combobox')).toHaveValue('base64');
    expect(nodesField(d.timeout)).toHaveValue(60);
  });

  it('emits each field change with the full config', async () => {
    const { user, last } = renderConfig(TelegramDownloadNodeConfig, {
      bot_token: 'tok',
      file_id: '',
      output_format: 'base64',
      timeout: 60,
    } as Config);

    fireEvent.change(screen.getByPlaceholderText('{{env.TELEGRAM_BOT_TOKEN}}'), { target: { value: '{{env.BOT}}' } });
    expect(last()).toEqual({ bot_token: '{{env.BOT}}', file_id: '', output_format: 'base64', timeout: 60 });

    await user.type(screen.getByPlaceholderText(d.fileIdPlaceholder), 'abc');
    expect(last().file_id).toBe('abc');

    await user.selectOptions(screen.getByRole('combobox'), 'url');
    expect(last().output_format).toBe('url');
    expect(screen.getByRole('option', { name: d.outputFormatUrl })).toBeInTheDocument();

    fireEvent.change(nodesField(d.timeout), { target: { value: '120' } });
    expect(last()).toEqual({ bot_token: '{{env.BOT}}', file_id: 'abc', output_format: 'url', timeout: 120 });
  });

  it('resyncs local state when the parent passes another node config', () => {
    const { setConfig } = renderConfig(TelegramDownloadNodeConfig, { bot_token: 'a', file_id: 'x' } as Config);
    act(() => setConfig({ bot_token: 'b', file_id: 'y', output_format: 'url', timeout: 5 }));
    expect(screen.getByPlaceholderText('{{env.TELEGRAM_BOT_TOKEN}}')).toHaveValue('b');
    expect(screen.getByPlaceholderText(d.fileIdPlaceholder)).toHaveValue('y');
    expect(screen.getByRole('combobox')).toHaveValue('url');
    expect(nodesField(d.timeout)).toHaveValue(5);
  });
});
