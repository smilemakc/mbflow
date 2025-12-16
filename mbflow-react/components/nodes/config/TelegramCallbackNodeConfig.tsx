/**
 * TelegramCallbackNodeConfig - React component for configuring Telegram callback query responses
 *
 * Features:
 * - Bot token configuration with variable support
 * - Callback query ID input
 * - Text response (optional)
 * - Show alert option
 * - URL for opening (optional)
 * - Cache time configuration
 * - Timeout configuration
 *
 * Usage:
 * ```tsx
 * <TelegramCallbackNodeConfigComponent
 *   config={callbackConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => updateNode(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from 'react';
import {VariableAutocomplete} from '@/components';
import {TelegramCallbackNodeConfig} from '@/types/nodeConfigs.ts';
import { useTranslation } from '@/store/translations';

interface Props {
    config: TelegramCallbackNodeConfig;
    nodeId?: string;
    onChange: (config: TelegramCallbackNodeConfig) => void;
}

export const TelegramCallbackNodeConfigComponent: React.FC<Props> = ({
                                                                         config,
                                                                         nodeId,
                                                                         onChange,
                                                                     }) => {
    const t = useTranslation();
    const [localConfig, setLocalConfig] = useState<TelegramCallbackNodeConfig>({
        bot_token: config.bot_token || '',
        callback_query_id: config.callback_query_id || '',
        text: config.text || '',
        show_alert: config.show_alert ?? false,
        url: config.url || '',
        cache_time: config.cache_time ?? 0,
        timeout: config.timeout,
    });

    useEffect(() => {
        if (JSON.stringify(config) !== JSON.stringify(localConfig)) {
            setLocalConfig({
                bot_token: config.bot_token || '',
                callback_query_id: config.callback_query_id || '',
                text: config.text || '',
                show_alert: config.show_alert ?? false,
                url: config.url || '',
                cache_time: config.cache_time ?? 0,
                timeout: config.timeout,
            });
        }
    }, [config]);

    const handleChange = (field: keyof TelegramCallbackNodeConfig, value: any) => {
        const newConfig = {...localConfig, [field]: value};
        setLocalConfig(newConfig);
        onChange(newConfig);
    };

    const inputClass =
        'w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200 placeholder-slate-400';
    const labelClass = 'text-xs font-semibold text-slate-600 dark:text-slate-400';
    const sectionClass = 'space-y-4 rounded-md border border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-slate-900/50 p-3';
    const sectionTitleClass = 'text-xs font-semibold uppercase text-gray-500 dark:text-slate-400';

    return (
        <div className="telegram-callback-config space-y-4">
            {/* API Credentials */}
            <div className={sectionClass}>
                <h4 className={sectionTitleClass}>{t.nodeConfig.telegramCallback.credentials}</h4>

                <div className="space-y-1.5">
                    <label className={labelClass}>{t.nodeConfig.telegramCallback.botToken}</label>
                    <VariableAutocomplete
                        type="input"
                        value={localConfig.bot_token}
                        onChange={(val) => handleChange('bot_token', val)}
                        placeholder="{{env.TELEGRAM_BOT_TOKEN}}"
                        className={inputClass}
                    />
                </div>
            </div>

            {/* Callback Settings */}
            <div className="space-y-4">
                <h4 className={sectionTitleClass}>{t.nodeConfig.telegramCallback.callbackQuery}</h4>

                <div className="space-y-1.5">
                    <label className={labelClass}>{t.nodeConfig.telegramCallback.callbackQueryId}</label>
                    <VariableAutocomplete
                        type="input"
                        value={localConfig.callback_query_id}
                        onChange={(val) => handleChange('callback_query_id', val)}
                        placeholder={t.nodeConfig.telegramCallback.callbackQueryIdPlaceholder}
                        className={inputClass}
                    />
                </div>

                <div className="space-y-1.5">
                    <label className={labelClass}>{t.nodeConfig.telegramCallback.textResponse}</label>
                    <VariableAutocomplete
                        type="textarea"
                        value={localConfig.text || ''}
                        onChange={(val) => handleChange('text', val)}
                        placeholder={t.nodeConfig.telegramCallback.textResponsePlaceholder}
                        rows={3}
                        className={inputClass + ' resize-none'}
                    />
                </div>

                <div className="flex items-center gap-2 cursor-pointer group">
                    <input
                        type="checkbox"
                        id="show-alert"
                        checked={localConfig.show_alert ?? false}
                        onChange={(e) => handleChange('show_alert', e.target.checked)}
                        className="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500/20 transition-colors"
                    />
                    <label
                        htmlFor="show-alert"
                        className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-slate-100 transition-colors cursor-pointer"
                    >
                        {t.nodeConfig.telegramCallback.showAlert}
                    </label>
                </div>
            </div>

            {/* Optional URL */}
            <div className="space-y-4">
                <h4 className={sectionTitleClass}>Optional</h4>

                <div className="space-y-1.5">
                    <label className={labelClass}>{t.nodeConfig.telegramCallback.url}</label>
                    <VariableAutocomplete
                        type="input"
                        value={localConfig.url || ''}
                        onChange={(val) => handleChange('url', val)}
                        placeholder={t.nodeConfig.telegramCallback.urlPlaceholder}
                        className={inputClass}
                    />
                </div>

                <div className="space-y-1.5">
                    <label className={labelClass}>{t.nodeConfig.telegramCallback.cacheTime}</label>
                    <input
                        type="number"
                        value={localConfig.cache_time ?? 0}
                        onChange={(e) =>
                            handleChange('cache_time', e.target.value ? parseInt(e.target.value, 10) : 0)
                        }
                        placeholder="0"
                        min="0"
                        max="86400"
                        className={inputClass}
                    />
                </div>

                <div className="space-y-1.5">
                    <label className={labelClass}>{t.nodeConfig.telegramCallback.timeout}</label>
                    <input
                        type="number"
                        value={localConfig.timeout ?? ''}
                        onChange={(e) =>
                            handleChange('timeout', e.target.value ? parseInt(e.target.value, 10) : undefined)
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

export default TelegramCallbackNodeConfigComponent;
