import React, {useEffect, useState} from 'react';
import type {TelegramDownloadNodeConfig as TelegramDownloadNodeConfigType} from '@/types/nodeConfigs.ts';
import {VariableAutocomplete} from '@/components';

interface Props {
    config: TelegramDownloadNodeConfigType;
    nodeId?: string;
    onChange: (config: TelegramDownloadNodeConfigType) => void;
}

export const TelegramDownloadNodeConfig: React.FC<Props> = ({
                                                                config,
                                                                nodeId,
                                                                onChange
                                                            }) => {
    const [localConfig, setLocalConfig] = useState<TelegramDownloadNodeConfigType>({
        bot_token: config.bot_token || '',
        file_id: config.file_id || '',
        output_format: config.output_format || 'base64',
        timeout: config.timeout || 60,
    });

    useEffect(() => {
        if (JSON.stringify(config) !== JSON.stringify(localConfig)) {
            setLocalConfig({
                bot_token: config.bot_token || '',
                file_id: config.file_id || '',
                output_format: config.output_format || 'base64',
                timeout: config.timeout || 60,
            });
        }
    }, [config]);

    const handleBotTokenChange = (value: string) => {
        const updated = {...localConfig, bot_token: value};
        setLocalConfig(updated);
        onChange(updated);
    };

    const handleFileIdChange = (value: string) => {
        const updated = {...localConfig, file_id: value};
        setLocalConfig(updated);
        onChange(updated);
    };

    const handleOutputFormatChange = (value: string) => {
        const updated = {...localConfig, output_format: value as 'base64' | 'url'};
        setLocalConfig(updated);
        onChange(updated);
    };

    const handleTimeoutChange = (value: number) => {
        const updated = {...localConfig, timeout: value};
        setLocalConfig(updated);
        onChange(updated);
    };

    const outputFormatOptions = [
        {label: 'Base64 (download content)', value: 'base64'},
        {label: 'URL (link only)', value: 'url'},
    ];

    return (
        <div className="space-y-4">
            {/* Credentials */}
            <div className="space-y-4 rounded-md border border-gray-200 bg-gray-50 p-3">
                <h4 className="text-xs font-semibold uppercase text-gray-500">Credentials</h4>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-gray-700">Bot Token</label>
                    <VariableAutocomplete
                        value={localConfig.bot_token}
                        onChange={handleBotTokenChange}
                        placeholder="{{env.TELEGRAM_BOT_TOKEN}}"
                        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                </div>
            </div>

            {/* File Settings */}
            <div className="space-y-4">
                <h4 className="text-xs font-semibold uppercase text-gray-500">File</h4>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-gray-700">File ID</label>
                    <VariableAutocomplete
                        value={localConfig.file_id}
                        onChange={handleFileIdChange}
                        placeholder="{{input.message.document.file_id}}"
                        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <p className="text-xs text-gray-500">
                        File ID from Telegram message (photo, document, audio, video, etc.)
                    </p>
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-gray-700">Output Format</label>
                    <select
                        value={localConfig.output_format}
                        onChange={(e) => handleOutputFormatChange(e.target.value)}
                        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    >
                        {outputFormatOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                                {option.label}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-gray-700">Timeout (seconds)</label>
                    <input
                        type="number"
                        min="1"
                        max="300"
                        value={localConfig.timeout}
                        onChange={(e) => handleTimeoutChange(Number(e.target.value))}
                        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                </div>
            </div>
        </div>
    );
};
