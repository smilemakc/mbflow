/**
 * TelegramNodeConfig - React component for configuring Telegram bot message nodes
 *
 * Ported from: /mbflow-ui/src/components/nodes/config/TelegramNodeConfig.vue
 *
 * Features:
 * - Bot token and chat ID configuration with variable support
 * - Multiple message types: text, photo, document, audio, video
 * - File source options: base64, URL, or file_id
 * - Parse mode selection: Markdown, MarkdownV2, HTML
 * - Message options: web page preview, notifications, content protection
 * - Variable autocomplete for all text inputs
 *
 * Usage:
 * ```tsx
 * <TelegramNodeConfigComponent
 *   config={telegramConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => updateNode(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from 'react';
import {VariableAutocomplete} from '../../builder/VariableAutocomplete.tsx';
import {
    TELEGRAM_FILE_SOURCES,
    TELEGRAM_MESSAGE_TYPES,
    TELEGRAM_PARSE_MODES,
    TelegramNodeConfig,
} from '@/types/nodeConfigs.ts';

interface Props {
    config: TelegramNodeConfig;
    nodeId?: string;
    onChange: (config: TelegramNodeConfig) => void;
}

export const TelegramNodeConfigComponent: React.FC<Props> = ({
                                                                 config,
                                                                 nodeId,
                                                                 onChange,
                                                             }) => {
    const [localConfig, setLocalConfig] = useState<TelegramNodeConfig>({
        bot_token: config.bot_token || '',
        chat_id: config.chat_id || '',
        message_type: config.message_type || 'text',
        text: config.text || '',
        parse_mode: config.parse_mode || 'HTML',
        file_source: config.file_source || 'url',
        file_data: config.file_data || '',
        file_name: config.file_name || '',
        disable_notification: config.disable_notification ?? false,
        protect_content: config.protect_content ?? false,
        disable_web_page_preview: config.disable_web_page_preview ?? false,
        timeout_seconds: config.timeout_seconds,
    });

    useEffect(() => {
        if (JSON.stringify(config) !== JSON.stringify(localConfig)) {
            setLocalConfig({
                bot_token: config.bot_token || '',
                chat_id: config.chat_id || '',
                message_type: config.message_type || 'text',
                text: config.text || '',
                parse_mode: config.parse_mode || 'HTML',
                file_source: config.file_source || 'url',
                file_data: config.file_data || '',
                file_name: config.file_name || '',
                disable_notification: config.disable_notification ?? false,
                protect_content: config.protect_content ?? false,
                disable_web_page_preview: config.disable_web_page_preview ?? false,
                timeout_seconds: config.timeout_seconds,
            });
        }
    }, [config]);

    const handleChange = (field: keyof TelegramNodeConfig, value: any) => {
        const newConfig = {...localConfig, [field]: value};
        setLocalConfig(newConfig);
        onChange(newConfig);
    };

    const inputClass =
        'w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200 placeholder-slate-400';
    const labelClass = 'text-xs font-semibold text-slate-600 dark:text-slate-400';
    const sectionClass = 'space-y-4 rounded-md border border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-slate-900/50 p-3';
    const sectionTitleClass = 'text-xs font-semibold uppercase text-gray-500 dark:text-slate-400';

    const getFileDataLabel = () => {
        switch (localConfig.file_source) {
            case 'url':
                return 'File URL';
            case 'file_id':
                return 'File ID';
            default:
                return 'Base64 Data';
        }
    };

    const getFileDataPlaceholder = () => {
        if (localConfig.file_source === 'url') {
            return 'https://example.com/image.jpg';
        }
        return 'File data...';
    };

    return (
        <div className="telegram-config space-y-4">
            {/* API Credentials */}
            <div className={sectionClass}>
                <h4 className={sectionTitleClass}>Credentials</h4>

                <div className="space-y-1.5">
                    <label className={labelClass}>Bot Token</label>
                    <VariableAutocomplete
                        type="input"
                        value={localConfig.bot_token}
                        onChange={(val) => handleChange('bot_token', val)}
                        placeholder="{{env.TELEGRAM_BOT_TOKEN}}"
                        className={inputClass}
                    />
                </div>

                <div className="space-y-1.5">
                    <label className={labelClass}>Chat ID</label>
                    <VariableAutocomplete
                        type="input"
                        value={localConfig.chat_id}
                        onChange={(val) => handleChange('chat_id', val)}
                        placeholder="@channel_name or {{env.CHAT_ID}}"
                        className={inputClass}
                    />
                </div>
            </div>

            {/* Message Settings */}
            <div className="space-y-4">
                <h4 className={sectionTitleClass}>Message</h4>

                <div className="space-y-1.5">
                    <label className={labelClass}>Message Type</label>
                    <select
                        value={localConfig.message_type}
                        onChange={(e) => handleChange('message_type', e.target.value as any)}
                        className={inputClass}
                    >
                        {TELEGRAM_MESSAGE_TYPES.map((type) => (
                            <option key={type} value={type}>
                                {type.charAt(0).toUpperCase() + type.slice(1)}
                            </option>
                        ))}
                    </select>
                </div>

                {/* Text Content */}
                {localConfig.message_type === 'text' ? (
                    <div className="space-y-1.5">
                        <label className={labelClass}>Message Text</label>
                        <VariableAutocomplete
                            type="textarea"
                            value={localConfig.text || ''}
                            onChange={(val) => handleChange('text', val)}
                            placeholder="Hello world! {{input.data}}"
                            rows={4}
                            className={inputClass + ' resize-none'}
                        />
                    </div>
                ) : (
                    <>
                        {/* Caption */}
                        <div className="space-y-1.5">
                            <label className={labelClass}>Caption (Optional)</label>
                            <VariableAutocomplete
                                type="textarea"
                                value={localConfig.text || ''}
                                onChange={(val) => handleChange('text', val)}
                                placeholder="Media caption..."
                                rows={2}
                                className={inputClass + ' resize-none'}
                            />
                        </div>

                        {/* File Settings */}
                        <div className={sectionClass}>
                            <h5 className="text-xs font-medium text-gray-700 dark:text-slate-300">
                                File Settings
                            </h5>

                            <div className="space-y-1.5">
                                <label className={labelClass}>File Source</label>
                                <select
                                    value={localConfig.file_source}
                                    onChange={(e) => handleChange('file_source', e.target.value as any)}
                                    className={inputClass}
                                >
                                    {TELEGRAM_FILE_SOURCES.map((source) => (
                                        <option key={source} value={source}>
                                            {source === 'file_id' ? 'File ID' : source.toUpperCase()}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            <div className="space-y-1.5">
                                <label className={labelClass}>{getFileDataLabel()}</label>
                                <VariableAutocomplete
                                    type="input"
                                    value={localConfig.file_data || ''}
                                    onChange={(val) => handleChange('file_data', val)}
                                    placeholder={getFileDataPlaceholder()}
                                    className={inputClass}
                                />
                            </div>

                            {(localConfig.file_source === 'base64' ||
                                localConfig.file_source === 'url') && (
                                <div className="space-y-1.5">
                                    <label className={labelClass}>File Name (Optional)</label>
                                    <VariableAutocomplete
                                        type="input"
                                        value={localConfig.file_name || ''}
                                        onChange={(val) => handleChange('file_name', val)}
                                        placeholder="image.jpg"
                                        className={inputClass}
                                    />
                                </div>
                            )}
                        </div>
                    </>
                )}

                {/* Common Options */}
                <div className="space-y-1.5">
                    <label className={labelClass}>Parse Mode</label>
                    <select
                        value={localConfig.parse_mode}
                        onChange={(e) => handleChange('parse_mode', e.target.value as any)}
                        className={inputClass}
                    >
                        {TELEGRAM_PARSE_MODES.map((mode) => (
                            <option key={mode} value={mode}>
                                {mode}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="flex flex-col gap-2 pt-2">
                    <label className="flex items-center gap-2 cursor-pointer group">
                        <input
                            type="checkbox"
                            checked={localConfig.disable_web_page_preview ?? false}
                            onChange={(e) => handleChange('disable_web_page_preview', e.target.checked)}
                            className="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500/20 transition-colors"
                        />
                        <span
                            className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-slate-100 transition-colors">
              Disable Web Page Preview
            </span>
                    </label>

                    <label className="flex items-center gap-2 cursor-pointer group">
                        <input
                            type="checkbox"
                            checked={localConfig.disable_notification ?? false}
                            onChange={(e) => handleChange('disable_notification', e.target.checked)}
                            className="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500/20 transition-colors"
                        />
                        <span
                            className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-slate-100 transition-colors">
              Disable Notification
            </span>
                    </label>

                    <label className="flex items-center gap-2 cursor-pointer group">
                        <input
                            type="checkbox"
                            checked={localConfig.protect_content ?? false}
                            onChange={(e) => handleChange('protect_content', e.target.checked)}
                            className="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500/20 transition-colors"
                        />
                        <span
                            className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-slate-100 transition-colors">
              Protect Content (Prevent Forwarding)
            </span>
                    </label>
                </div>

                {/* Timeout */}
                <div className="space-y-1.5">
                    <label className={labelClass}>Timeout (seconds)</label>
                    <input
                        type="number"
                        value={localConfig.timeout_seconds ?? ''}
                        onChange={(e) =>
                            handleChange(
                                'timeout_seconds',
                                e.target.value ? parseInt(e.target.value, 10) : undefined
                            )
                        }
                        placeholder="30"
                        min="1"
                        className={inputClass}
                    />
                </div>
            </div>
        </div>
    );
};

export default TelegramNodeConfigComponent;
