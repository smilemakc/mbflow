import React, {useEffect, useState} from 'react';
import {DEFAULT_TOOL_CALL_CONFIG, LLM_PROVIDER_MODELS, LLMNodeConfig} from '@/types/nodeConfigs.ts';
import {VariableAutocomplete} from '@/components';
import {Button} from '@/components/ui';
import {useTranslation} from '@/store/translations';

interface Props {
    config: LLMNodeConfig;
    nodeId?: string;
    onChange: (config: LLMNodeConfig) => void;
}

export const LLMNodeConfigComponent: React.FC<Props> = ({config, nodeId, onChange}) => {
    const t = useTranslation();
    const [localConfig, setLocalConfig] = useState<LLMNodeConfig>({...config});
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [toolCallingEnabled, setToolCallingEnabled] = useState(!!localConfig.tool_call_config);
    const [toolCallMode, setToolCallMode] = useState<'auto' | 'manual'>(
        localConfig.tool_call_config?.mode || 'manual'
    );
    const [stopSequencesText, setStopSequencesText] = useState(
        (localConfig.stop_sequences || []).join('\n')
    );

    const availableModels = LLM_PROVIDER_MODELS[localConfig.provider] || [];

    const updateConfig = (updates: Partial<LLMNodeConfig>) => {
        const newConfig = {...localConfig, ...updates};
        setLocalConfig(newConfig);
        onChange(newConfig);
    };

    const handleProviderChange = (provider: LLMNodeConfig['provider']) => {
        const models = LLM_PROVIDER_MODELS[provider];
        const updates: Partial<LLMNodeConfig> = {provider};

        if (models && models.length > 0 && !models.includes(localConfig.model)) {
            updates.model = models[0];
        }

        updateConfig(updates);
    };

    const handleToolCallingEnabledChange = (enabled: boolean) => {
        setToolCallingEnabled(enabled);

        if (enabled && !localConfig.tool_call_config) {
            updateConfig({
                tool_call_config: {...DEFAULT_TOOL_CALL_CONFIG},
                functions: [],
            });
        } else if (!enabled) {
            updateConfig({
                tool_call_config: undefined,
                functions: undefined,
            });
        }
    };

    const handleToolCallModeChange = (mode: 'auto' | 'manual') => {
        setToolCallMode(mode);

        if (localConfig.tool_call_config) {
            updateConfig({
                tool_call_config: {
                    ...localConfig.tool_call_config,
                    mode,
                },
            });
        }
    };

    const handleToolCallConfigChange = (updates: Partial<typeof DEFAULT_TOOL_CALL_CONFIG>) => {
        if (localConfig.tool_call_config) {
            updateConfig({
                tool_call_config: {
                    ...localConfig.tool_call_config,
                    ...updates,
                },
            });
        }
    };

    const handleStopSequencesChange = (text: string) => {
        setStopSequencesText(text);

        const sequences = text
            .split('\n')
            .map((s) => s.trim())
            .filter((s) => s.length > 0);

        updateConfig({stop_sequences: sequences});
    };

    useEffect(() => {
        if (JSON.stringify(config) !== JSON.stringify(localConfig)) {
            setLocalConfig({...config});
            setToolCallingEnabled(!!config.tool_call_config);
            setToolCallMode(config.tool_call_config?.mode || 'manual');
            setStopSequencesText((config.stop_sequences || []).join('\n'));
        }
    }, [config]);

    const inputClasses = 'w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors';
    const selectClasses = 'w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors';
    const textareaClasses = 'w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors font-mono resize-y';
    const labelClasses = 'text-sm font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-2';
    const hintClasses = 'text-xs font-normal text-slate-500 dark:text-slate-400';

    const functionCount = localConfig.functions?.length || 0;

    return (
        <div className="flex flex-col gap-4">
            {/* Basic Settings */}
            <div className="flex flex-col gap-1.5">
                <label className={labelClasses}>{t.nodeConfig.llm.provider}</label>
                <select
                    value={localConfig.provider}
                    onChange={(e) => handleProviderChange(e.target.value as LLMNodeConfig['provider'])}
                    className={selectClasses}
                >
                    <option value="openai">OpenAI</option>
                    <option value="anthropic">Anthropic</option>
                    <option value="google">Google</option>
                    <option value="azure">Azure</option>
                    <option value="ollama">Ollama</option>
                </select>
            </div>

            <div className="flex flex-col gap-1.5">
                <label className={labelClasses}>{t.nodeConfig.llm.model}</label>
                <select
                    value={localConfig.model}
                    onChange={(e) => updateConfig({model: e.target.value})}
                    className={selectClasses}
                >
                    {availableModels.map((model) => (
                        <option key={model} value={model}>
                            {model}
                        </option>
                    ))}
                </select>
            </div>

            <div className="flex flex-col gap-1.5">
                <label className={labelClasses}>{t.nodeConfig.llm.apiKey}</label>
                <VariableAutocomplete
                    value={localConfig.api_key}
                    onChange={(value) => updateConfig({api_key: value})}
                    placeholder={t.nodeConfig.llm.apiKeyPlaceholder}
                    className={inputClasses}
                    type="input"
                />
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {t.nodeConfig.llm.apiKeyHint}
                </p>
            </div>

            <div className="flex flex-col gap-1.5">
                <label className={labelClasses}>{t.nodeConfig.llm.systemPrompt}</label>
                <VariableAutocomplete
                    value={localConfig.instruction || ''}
                    onChange={(value) => updateConfig({instruction: value})}
                    placeholder={t.nodeConfig.llm.systemPromptPlaceholder}
                    className={textareaClasses}
                    type="textarea"
                    rows={3}
                />
            </div>

            <div className="flex flex-col gap-1.5">
                <label className={labelClasses}>{t.nodeConfig.llm.userPrompt}</label>
                <VariableAutocomplete
                    value={localConfig.prompt}
                    onChange={(value) => updateConfig({prompt: value})}
                    placeholder={t.nodeConfig.llm.userPromptPlaceholder}
                    className={textareaClasses}
                    type="textarea"
                    rows={5}
                />
            </div>

            {/* Advanced Settings Toggle */}
            <Button
                onClick={() => setShowAdvanced(!showAdvanced)}
                variant="outline"
                size="sm"
                type="button"
            >
                {showAdvanced ? '▼' : '▶'} {t.nodeConfig.llm.advancedSettings}
            </Button>

            {/* Advanced Section */}
            {showAdvanced && (
                <div
                    className="flex flex-col gap-4 p-4 bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-md">
                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>
                            {t.nodeConfig.llm.temperature}
                            <span className={hintClasses}>{t.nodeConfig.llm.temperatureHint}</span>
                        </label>
                        <input
                            type="number"
                            min="0"
                            max="2"
                            step="0.1"
                            value={localConfig.temperature || 0.7}
                            onChange={(e) => updateConfig({temperature: parseFloat(e.target.value)})}
                            className={inputClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>
                            {t.nodeConfig.llm.maxTokens}
                            <span className={hintClasses}>{t.nodeConfig.llm.maxTokensHint}</span>
                        </label>
                        <input
                            type="number"
                            min="1"
                            max="100000"
                            value={localConfig.max_tokens || 1000}
                            onChange={(e) => updateConfig({max_tokens: parseInt(e.target.value, 10)})}
                            className={inputClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>
                            {t.nodeConfig.llm.topP}
                            <span className={hintClasses}>{t.nodeConfig.llm.topPHint}</span>
                        </label>
                        <input
                            type="number"
                            min="0"
                            max="1"
                            step="0.1"
                            value={localConfig.top_p !== undefined ? localConfig.top_p : 1}
                            onChange={(e) => updateConfig({top_p: parseFloat(e.target.value)})}
                            className={inputClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>
                            {t.nodeConfig.llm.frequencyPenalty}
                            <span className={hintClasses}>{t.nodeConfig.llm.frequencyPenaltyHint}</span>
                        </label>
                        <input
                            type="number"
                            min="-2"
                            max="2"
                            step="0.1"
                            value={localConfig.frequency_penalty !== undefined ? localConfig.frequency_penalty : 0}
                            onChange={(e) => updateConfig({frequency_penalty: parseFloat(e.target.value)})}
                            className={inputClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>
                            {t.nodeConfig.llm.presencePenalty}
                            <span className={hintClasses}>{t.nodeConfig.llm.presencePenaltyHint}</span>
                        </label>
                        <input
                            type="number"
                            min="-2"
                            max="2"
                            step="0.1"
                            value={localConfig.presence_penalty !== undefined ? localConfig.presence_penalty : 0}
                            onChange={(e) => updateConfig({presence_penalty: parseFloat(e.target.value)})}
                            className={inputClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>{t.nodeConfig.llm.stopSequences}</label>
                        <textarea
                            value={stopSequencesText}
                            onChange={(e) => handleStopSequencesChange(e.target.value)}
                            placeholder={t.nodeConfig.llm.stopSequencesPlaceholder}
                            rows={3}
                            className={textareaClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>{t.nodeConfig.llm.responseFormat}</label>
                        <select
                            value={localConfig.response_format || 'text'}
                            onChange={(e) => updateConfig({response_format: e.target.value as 'text' | 'json'})}
                            className={selectClasses}
                        >
                            <option value="text">{t.nodeConfig.llm.responseFormatText}</option>
                            <option value="json">{t.nodeConfig.llm.responseFormatJson}</option>
                        </select>
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>{t.nodeConfig.llm.timeout}</label>
                        <input
                            type="number"
                            min="1"
                            max="300"
                            value={localConfig.timeout_seconds || 30}
                            onChange={(e) => updateConfig({timeout_seconds: parseInt(e.target.value, 10)})}
                            className={inputClasses}
                        />
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className={labelClasses}>{t.nodeConfig.llm.retryCount}</label>
                        <input
                            type="number"
                            min="0"
                            max="5"
                            value={localConfig.retry_count !== undefined ? localConfig.retry_count : 0}
                            onChange={(e) => updateConfig({retry_count: parseInt(e.target.value, 10)})}
                            className={inputClasses}
                        />
                    </div>

                    {/* Tool Calling Section */}
                    <div className="mt-4 pt-4 border-t-2 border-slate-200 dark:border-slate-700">
                        <h3 className="text-sm font-bold text-slate-700 dark:text-slate-300 mb-3">
                            🔧 Tool Calling (Phase 1)
                        </h3>

                        <div className="flex flex-col gap-1.5 mb-4">
                            <label
                                className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300 cursor-pointer select-none">
                                <input
                                    type="checkbox"
                                    checked={toolCallingEnabled}
                                    onChange={(e) => handleToolCallingEnabledChange(e.target.checked)}
                                    className="w-4 h-4 cursor-pointer accent-blue-500"
                                />
                                Enable Tool Calling
                            </label>
                        </div>

                        {toolCallingEnabled && (
                            <div
                                className="mt-3 p-4 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-md flex flex-col gap-4">
                                <div className="flex flex-col gap-1.5">
                                    <label className={labelClasses}>Tool Call Mode</label>
                                    <select
                                        value={toolCallMode}
                                        onChange={(e) => handleToolCallModeChange(e.target.value as 'auto' | 'manual')}
                                        className={selectClasses}
                                    >
                                        <option value="auto">Auto (Automatic loop until completion)</option>
                                        <option value="manual">Manual (Connect FunctionCall nodes via edges)</option>
                                    </select>
                                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                        <strong>Auto:</strong> LLM automatically calls functions in a loop.{' '}
                                        <strong>Manual:</strong> Use FunctionCall nodes connected via edges.
                                    </p>
                                </div>

                                {toolCallMode === 'auto' && (
                                    <div
                                        className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md flex flex-col gap-3">
                                        <div className="flex flex-col gap-1.5">
                                            <label className={labelClasses}>
                                                Max Iterations
                                                <span className={hintClasses}>(Prevents infinite loops)</span>
                                            </label>
                                            <input
                                                type="number"
                                                min="1"
                                                max="50"
                                                value={localConfig.tool_call_config?.max_iterations || 10}
                                                onChange={(e) =>
                                                    handleToolCallConfigChange({max_iterations: parseInt(e.target.value, 10)})
                                                }
                                                className={inputClasses}
                                            />
                                        </div>

                                        <div className="flex flex-col gap-1.5">
                                            <label className={labelClasses}>
                                                Timeout Per Tool (seconds)
                                                <span className={hintClasses}>(Max time for each tool call)</span>
                                            </label>
                                            <input
                                                type="number"
                                                min="1"
                                                max="300"
                                                value={localConfig.tool_call_config?.timeout_per_tool || 30}
                                                onChange={(e) =>
                                                    handleToolCallConfigChange({timeout_per_tool: parseInt(e.target.value, 10)})
                                                }
                                                className={inputClasses}
                                            />
                                        </div>

                                        <div className="flex flex-col gap-1.5">
                                            <label className={labelClasses}>
                                                Total Timeout (seconds)
                                                <span className={hintClasses}>(Max time for entire loop)</span>
                                            </label>
                                            <input
                                                type="number"
                                                min="1"
                                                max="1800"
                                                value={localConfig.tool_call_config?.total_timeout || 300}
                                                onChange={(e) =>
                                                    handleToolCallConfigChange({total_timeout: parseInt(e.target.value, 10)})
                                                }
                                                className={inputClasses}
                                            />
                                        </div>

                                        <div className="flex flex-col gap-1.5">
                                            <label
                                                className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300 cursor-pointer select-none">
                                                <input
                                                    type="checkbox"
                                                    checked={localConfig.tool_call_config?.stop_on_tool_failure || false}
                                                    onChange={(e) =>
                                                        handleToolCallConfigChange({stop_on_tool_failure: e.target.checked})
                                                    }
                                                    className="w-4 h-4 cursor-pointer accent-blue-500"
                                                />
                                                Stop on tool failure
                                            </label>
                                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                                If enabled, execution stops when any tool fails. Otherwise, errors are
                                                added to conversation.
                                            </p>
                                        </div>
                                    </div>
                                )}

                                <div className="flex flex-col gap-1.5">
                                    <label className={labelClasses}>
                                        Functions ({functionCount})
                                        <span className={hintClasses}>Phase 1: Built-in functions only</span>
                                    </label>
                                    <div
                                        className="p-6 bg-slate-50 dark:bg-slate-800 border-2 border-dashed border-slate-300 dark:border-slate-600 rounded-lg text-center">
                                        <p className="text-sm text-slate-600 dark:text-slate-400">
                                            Function editor will be available in the next update.
                                        </p>
                                        <p className="text-xs text-slate-500 dark:text-slate-500 mt-2">
                                            For now, configure functions via JSON in workflow definition.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
};
