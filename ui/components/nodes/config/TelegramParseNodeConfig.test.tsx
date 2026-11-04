import { describe, expect, it } from 'vitest';
import { act, screen } from '@testing-library/react';
import { TelegramParseNodeConfigComponent } from './TelegramParseNodeConfig';
import type { TelegramParseNodeConfig } from '@/types/nodeConfigs';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';

describe('TelegramParseNodeConfig', () => {
  const t = nodesEnT();
  const p = t.nodeConfig.telegramParse;

  it('fills defaults into an empty config and emits them once', () => {
    const { onChange } = renderConfig(TelegramParseNodeConfigComponent, {} as TelegramParseNodeConfig);
    expect(nodesField(p.extractFiles)).toBeChecked();
    expect(nodesField(p.extractCommands)).toBeChecked();
    expect(nodesField(p.extractEntities)).not.toBeChecked();
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenLastCalledWith({ extract_files: true, extract_commands: true, extract_entities: false });
    expect(screen.getByText(p.infoText)).toBeInTheDocument();
  });

  it('does not emit when the config is already complete', () => {
    const { onChange } = renderConfig(TelegramParseNodeConfigComponent, {
      extract_files: false,
      extract_commands: true,
      extract_entities: true,
    });
    expect(onChange).not.toHaveBeenCalled();
    expect(nodesField(p.extractFiles)).not.toBeChecked();
    expect(nodesField(p.extractEntities)).toBeChecked();
  });

  it('toggles each option', async () => {
    const { user, last } = renderConfig(TelegramParseNodeConfigComponent, {
      extract_files: true,
      extract_commands: true,
      extract_entities: false,
    });
    await user.click(nodesField(p.extractFiles));
    expect(last()).toEqual({ extract_files: false, extract_commands: true, extract_entities: false });
    await user.click(nodesField(p.extractCommands));
    expect(last()).toEqual({ extract_files: false, extract_commands: false, extract_entities: false });
    await user.click(nodesField(p.extractEntities));
    expect(last()).toEqual({ extract_files: false, extract_commands: false, extract_entities: true });
  });

  it('syncs when the parent replaces the config', () => {
    const { setConfig } = renderConfig(TelegramParseNodeConfigComponent, {
      extract_files: true,
      extract_commands: true,
      extract_entities: false,
    });
    act(() => setConfig({ extract_files: false, extract_commands: false, extract_entities: true }));
    expect(nodesField(p.extractFiles)).not.toBeChecked();
    expect(nodesField(p.extractCommands)).not.toBeChecked();
    expect(nodesField(p.extractEntities)).toBeChecked();
  });
});
