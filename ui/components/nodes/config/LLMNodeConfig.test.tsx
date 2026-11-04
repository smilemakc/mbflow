import { afterEach, describe, expect, it } from 'vitest';
import { act, fireEvent, screen } from '@testing-library/react';
import { LLMNodeConfigComponent } from './LLMNodeConfig';
import { DEFAULT_TOOL_CALL_CONFIG, LLM_PROVIDER_MODELS, type LLMNodeConfig } from '@/types/nodeConfigs';
import type { ApiMock } from '@/test/apiMock';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';
import { nodesApiClient, nodesMockApiClients, nodesOperation } from '@/test/nodesApiClients';

const base: LLMNodeConfig = {
  provider: 'openai',
  model: LLM_PROVIDER_MODELS.openai[1],
  api_key: '',
  prompt: '',
};

let api: ApiMock | undefined;
afterEach(() => api?.restore());

describe('LLMNodeConfig', () => {
  const t = nodesEnT();
  const l = t.nodeConfig.llm;
  const openAdvanced = async (user: { click: (el: Element) => Promise<void> }) =>
    user.click(screen.getByRole('button', { name: new RegExp(l.advancedSettings) }));

  it('edits basic settings and picks the first model of a new provider', async () => {
    const { user, last } = renderConfig(LLMNodeConfigComponent, base);
    const model = nodesField<HTMLSelectElement>(l.model);
    expect(Array.from(model.options).map((o) => o.value)).toEqual(LLM_PROVIDER_MODELS.openai);

    await user.selectOptions(model, LLM_PROVIDER_MODELS.openai[2]);
    expect(last().model).toBe(LLM_PROVIDER_MODELS.openai[2]);

    await user.selectOptions(nodesField<HTMLSelectElement>(l.provider), 'anthropic');
    expect(last()).toMatchObject({ provider: 'anthropic', model: LLM_PROVIDER_MODELS.anthropic[0] });

    // gpt-4.1 exists for both OpenAI APIs, so it is kept when switching between them.
    await user.selectOptions(nodesField<HTMLSelectElement>(l.provider), 'openai');
    await user.selectOptions(nodesField<HTMLSelectElement>(l.model), 'gpt-4.1');
    await user.selectOptions(nodesField<HTMLSelectElement>(l.provider), 'openai-responses');
    expect(last()).toMatchObject({ provider: 'openai-responses', model: 'gpt-4.1' });

    fireEvent.change(screen.getByPlaceholderText(l.apiKeyPlaceholder), { target: { value: '{{env.KEY}}' } });
    expect(last().api_key).toBe('{{env.KEY}}');
    await user.type(screen.getByPlaceholderText(l.systemPromptPlaceholder), 'Be brief');
    expect(last().instruction).toBe('Be brief');
    await user.type(screen.getByPlaceholderText(l.userPromptPlaceholder), 'Hi');
    expect(last().prompt).toBe('Hi');
  });

  it('advanced settings are hidden until toggled and emit parsed numbers', async () => {
    const { user, last } = renderConfig(LLMNodeConfigComponent, base);
    expect(screen.queryByText(l.temperature)).toBeNull();
    await openAdvanced(user);

    expect(nodesField(l.temperature)).toHaveValue(0.7);
    expect(nodesField(l.maxTokens)).toHaveValue(1000);
    expect(nodesField(l.topP)).toHaveValue(1);
    expect(nodesField(l.frequencyPenalty)).toHaveValue(0);
    expect(nodesField(l.presencePenalty)).toHaveValue(0);
    expect(nodesField(l.timeout)).toHaveValue(30);
    expect(nodesField(l.retryCount)).toHaveValue(0);

    fireEvent.change(nodesField(l.temperature), { target: { value: '1.2' } });
    expect(last().temperature).toBe(1.2);
    fireEvent.change(nodesField(l.maxTokens), { target: { value: '256' } });
    expect(last().max_tokens).toBe(256);
    fireEvent.change(nodesField(l.topP), { target: { value: '0.5' } });
    expect(last().top_p).toBe(0.5);
    fireEvent.change(nodesField(l.frequencyPenalty), { target: { value: '-1' } });
    expect(last().frequency_penalty).toBe(-1);
    fireEvent.change(nodesField(l.presencePenalty), { target: { value: '0.3' } });
    expect(last().presence_penalty).toBe(0.3);
    fireEvent.change(nodesField(l.timeout), { target: { value: '90' } });
    expect(last().timeout_seconds).toBe(90);
    fireEvent.change(nodesField(l.retryCount), { target: { value: '2' } });
    expect(last().retry_count).toBe(2);

    fireEvent.change(screen.getByPlaceholderText(l.stopSequencesPlaceholder), { target: { value: 'END\n\n  STOP  \n' } });
    expect(last().stop_sequences).toEqual(['END', 'STOP']);

    await user.selectOptions(nodesField<HTMLSelectElement>(l.responseFormat), 'json');
    expect(last().response_format).toBe('json');

    await openAdvanced(user);
    expect(screen.queryByText(l.temperature)).toBeNull();
  });

  it('enables manual tool calling by default and removes it when disabled', async () => {
    const { user, last } = renderConfig(LLMNodeConfigComponent, base);
    await openAdvanced(user);
    const toggle = screen.getByRole('checkbox', { name: 'Enable Tool Calling' });
    await user.click(toggle);
    expect(last().tool_call_config).toEqual(DEFAULT_TOOL_CALL_CONFIG);
    expect(last().functions).toEqual([]);
    expect(screen.getByText(l.apiTools.autoOnly)).toBeInTheDocument();
    expect(screen.queryByText('Max Iterations')).toBeNull();

    await user.click(toggle);
    expect(last()).toMatchObject({ tool_call_config: undefined, functions: undefined, api_tools: undefined });
    expect(screen.queryByText('Tool Call Mode')).toBeNull();
  });

  it('auto mode exposes loop limits and the API tools picker', async () => {
    api = nodesMockApiClients([nodesApiClient('c1', [nodesOperation('op1', { summary: 'Op one' })])]);
    const { user, last } = renderConfig(
      LLMNodeConfigComponent,
      {
        ...base,
        tool_call_config: { ...DEFAULT_TOOL_CALL_CONFIG },
        functions: [{ type: 'builtin', name: 'now', description: 'time', builtin_name: 'get_current_time' }],
      },
      { router: true },
    );
    await openAdvanced(user);
    expect(screen.getByText(l.apiTools.otherFunctions.replace('{count}', '1'))).toBeInTheDocument();

    await user.selectOptions(nodesField<HTMLSelectElement>('Tool Call Mode'), 'auto');
    expect(last().tool_call_config?.mode).toBe('auto');

    fireEvent.change(nodesField('Max Iterations'), { target: { value: '5' } });
    expect(last().tool_call_config?.max_iterations).toBe(5);
    fireEvent.change(nodesField('Timeout Per Tool'), { target: { value: '12' } });
    expect(last().tool_call_config?.timeout_per_tool).toBe(12);
    fireEvent.change(nodesField('Total Timeout'), { target: { value: '600' } });
    expect(last().tool_call_config?.total_timeout).toBe(600);
    await user.click(screen.getByRole('checkbox', { name: 'Stop on tool failure' }));
    expect(last().tool_call_config?.stop_on_tool_failure).toBe(true);

    await user.selectOptions(await screen.findByRole('combobox', { name: l.apiTools.addClient }), 'c1');
    await user.click(await screen.findByRole('checkbox', { name: /Op one/ }));
    expect(last().api_tools).toEqual([{ client_id: 'c1', operation_ids: ['op1'] }]);
    expect(last().tool_call_config).toMatchObject({ mode: 'auto', max_iterations: 5, stop_on_tool_failure: true });
  });

  it('resyncs tool calling state and stop sequences from the parent', async () => {
    const { user, setConfig } = renderConfig(LLMNodeConfigComponent, base, { router: true });
    await openAdvanced(user);
    api = nodesMockApiClients([]);
    act(() =>
      setConfig({
        ...base,
        stop_sequences: ['A', 'B'],
        tool_call_config: { ...DEFAULT_TOOL_CALL_CONFIG, mode: 'auto' },
      }),
    );
    expect(screen.getByPlaceholderText(l.stopSequencesPlaceholder)).toHaveValue('A\nB');
    expect(screen.getByRole('checkbox', { name: 'Enable Tool Calling' })).toBeChecked();
    expect(nodesField<HTMLSelectElement>('Tool Call Mode')).toHaveValue('auto');
    expect(await screen.findByText(l.apiTools.noClients)).toBeInTheDocument();
  });
});
