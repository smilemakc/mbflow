/**
 * TelegramParseNodeConfig - React component for configuring Telegram parse nodes
 *
 * Features:
 * - Extract files option: checkbox
 * - Extract commands option: checkbox
 * - Extract entities option: checkbox
 *
 * Usage:
 * ```tsx
 * <TelegramParseNodeConfigComponent
 *   config={parseConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => updateNode(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from 'react';
import {TelegramParseNodeConfig} from '@/types/nodeConfigs.ts';

interface TelegramParseNodeConfigProps {
    config: TelegramParseNodeConfig;
    nodeId?: string;
    onChange: (config: TelegramParseNodeConfig) => void;
}

export const TelegramParseNodeConfigComponent: React.FC<
    TelegramParseNodeConfigProps
> = ({config, nodeId, onChange}) => {
    const [localConfig, setLocalConfig] = useState<TelegramParseNodeConfig>({
        extract_files: config.extract_files ?? true,
        extract_commands: config.extract_commands ?? true,
        extract_entities: config.extract_entities ?? false,
    });

    useEffect(() => {
        const newConfig: TelegramParseNodeConfig = {
            extract_files: config.extract_files ?? true,
            extract_commands: config.extract_commands ?? true,
            extract_entities: config.extract_entities ?? false,
        };

        if (JSON.stringify(newConfig) !== JSON.stringify(localConfig)) {
            setLocalConfig(newConfig);
        }
    }, [config]);

    useEffect(() => {
        if (JSON.stringify(localConfig) !== JSON.stringify(config)) {
            onChange(localConfig);
        }
    }, [localConfig]);

    const handleChange = (field: keyof TelegramParseNodeConfig, value: boolean) => {
        setLocalConfig((prev) => ({...prev, [field]: value}));
    };

    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-3">
                <h3 className="text-sm font-semibold text-gray-700">Extract Options</h3>

                <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={localConfig.extract_files ?? true}
                        onChange={(e) => handleChange('extract_files', e.target.checked)}
                        className="w-[18px] h-[18px] cursor-pointer"
                    />
                    <span>Extract Files</span>
                    <span className="text-xs text-gray-500">
            (documents, photos, videos, audio)
          </span>
                </label>

                <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={localConfig.extract_commands ?? true}
                        onChange={(e) => handleChange('extract_commands', e.target.checked)}
                        className="w-[18px] h-[18px] cursor-pointer"
                    />
                    <span>Extract Commands</span>
                    <span className="text-xs text-gray-500">(e.g., /start, /help)</span>
                </label>

                <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={localConfig.extract_entities ?? false}
                        onChange={(e) => handleChange('extract_entities', e.target.checked)}
                        className="w-[18px] h-[18px] cursor-pointer"
                    />
                    <span>Extract Entities</span>
                    <span className="text-xs text-gray-500">
            (mentions, hashtags, URLs)
          </span>
                </label>
            </div>

            <div className="p-3 bg-blue-50 border border-blue-200 rounded-md">
                <p className="text-xs text-blue-900">
                    Selected options control what data is extracted from Telegram updates
                    and made available in the node output.
                </p>
            </div>
        </div>
    );
};

export default TelegramParseNodeConfigComponent;
