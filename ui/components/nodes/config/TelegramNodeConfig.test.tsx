import { describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { TelegramNodeConfigComponent } from './TelegramNodeConfig';
import type { TelegramNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

const base: TelegramNodeConfig = {
  bot_token: 'tok',
  chat_id: '42',
  message_type: 'text',
  text: '',
  parse_mode: 'HTML',
};

describe('TelegramNodeConfig', () => {
  const t = nodesEnT();
  const tg = t.nodeConfig.telegram;

  it('edits credentials and text messages', async () => {
    const { user, last } = renderConfig(TelegramNodeConfigComponent, base);
    expect(screen.getByPlaceholderText(tg.messageTextPlaceholder)).toBeInTheDocument();
    expect(screen.queryByText(tg.fileSettings)).toBeNull();

    fireEvent.change(screen.getByPlaceholderText(tg.botTokenPlaceholder), { target: { value: 'new' } });
    expect(last()).toMatchObject({ bot_token: 'new', chat_id: '42', file_source: 'url' });

    await user.type(screen.getByPlaceholderText(tg.chatIdPlaceholder), '7');
    expect(last().chat_id).toBe('427');

    await user.type(screen.getByPlaceholderText(tg.messageTextPlaceholder), 'Hello');
    expect(last().text).toBe('Hello');

    await user.selectOptions(nodesField<HTMLSelectElement>(tg.parseMode), 'MarkdownV2');
    expect(last().parse_mode).toBe('MarkdownV2');
  });

  it('shows file settings for media messages and labels the file field by source', async () => {
    const { user, last } = renderConfig(TelegramNodeConfigComponent, base);
    await user.selectOptions(nodesField<HTMLSelectElement>(tg.messageType), 'photo');
    expect(last().message_type).toBe('photo');
    expect(screen.getByText(tg.fileSettings)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(tg.captionPlaceholder)).toBeInTheDocument();

    // url source: URL label, url placeholder and file name field
    expect(screen.getByText(tg.fileData)).toBeInTheDocument();
    const fileData = screen.getByPlaceholderText(tg.fileDataPlaceholder);
    fireEvent.change(fileData, { target: { value: 'https://img' } });
    expect(last().file_data).toBe('https://img');
    await user.type(screen.getByPlaceholderText(tg.fileNamePlaceholder), 'a.png');
    expect(last().file_name).toBe('a.png');

    await user.type(screen.getByPlaceholderText(tg.captionPlaceholder), 'Cap');
    expect(last().text).toBe('Cap');

    const source = nodesField<HTMLSelectElement>(tg.fileSource);
    await user.selectOptions(source, 'file_id');
    expect(last().file_source).toBe('file_id');
    expect(screen.getByText(tg.fileId, { selector: 'label' })).toBeInTheDocument();
    expect(screen.getByPlaceholderText('File data...')).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(tg.fileNamePlaceholder)).toBeNull();

    await user.selectOptions(source, 'base64');
    expect(screen.getByText(tg.base64Data)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(tg.fileNamePlaceholder)).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'File ID' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'BASE64' })).toBeInTheDocument();
  });

  it('toggles message options and parses timeout', async () => {
    const { user, last } = renderConfig(TelegramNodeConfigComponent, base);
    await user.click(nodesField(tg.disableWebPagePreview));
    expect(last().disable_web_page_preview).toBe(true);
    await user.click(nodesField(tg.disableNotification));
    expect(last().disable_notification).toBe(true);
    await user.click(nodesField(tg.protectContent));
    expect(last().protect_content).toBe(true);

    const timeout = nodesField(tg.timeout);
    fireEvent.change(timeout, { target: { value: '15' } });
    expect(last().timeout_seconds).toBe(15);
    fireEvent.change(timeout, { target: { value: '' } });
    expect(last().timeout_seconds).toBeUndefined();
  });

  it('applies defaults for an empty config and resyncs from the parent', () => {
    const { setConfig } = renderConfig(TelegramNodeConfigComponent, {} as TelegramNodeConfig);
    expect(nodesField<HTMLSelectElement>(tg.messageType)).toHaveValue('text');
    expect(nodesField<HTMLSelectElement>(tg.parseMode)).toHaveValue('HTML');
    act(() => setConfig({ ...base, message_type: 'document', file_source: 'base64', parse_mode: 'Markdown', protect_content: true }));
    expect(nodesField<HTMLSelectElement>(tg.messageType)).toHaveValue('document');
    expect(nodesField<HTMLSelectElement>(tg.fileSource)).toHaveValue('base64');
    expect(nodesField(tg.protectContent)).toBeChecked();
  });
});
