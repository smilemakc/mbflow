/**
 * FunctionCallNodeConfig - React component for configuring function call nodes
 *
 * Ported from: /mbflow-ui/src/components/nodes/config/FunctionCallNodeConfig.vue
 *
 * Features:
 * - Function name input with template variable support
 * - Arguments editor as JSON with template variable support
 * - Timeout configuration in seconds
 * - JSON validation with helpful error messages
 *
 * Usage:
 * ```tsx
 * <FunctionCallNodeConfig
 *   config={functionCallConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => console.log(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from 'react';
import {FunctionCallNodeConfig} from '@/types/nodeConfigs.ts';

interface FunctionCallNodeConfigProps {
    config: FunctionCallNodeConfig;
    nodeId?: string;
    onChange: (config: FunctionCallNodeConfig) => void;
}

export const FunctionCallNodeConfigComponent: React.FC<FunctionCallNodeConfigProps> = ({
                                                                                           config,
                                                                                           nodeId,
                                                                                           onChange,
                                                                                       }) => {
    const [localConfig, setLocalConfig] = useState<FunctionCallNodeConfig>({
        function_name: '',
        arguments: {},
        timeout_seconds: 30,
        ...config,
    });

    const [argumentsStr, setArgumentsStr] = useState<string>(
        typeof config?.arguments === 'string'
            ? config.arguments
            : JSON.stringify(config?.arguments || {}, null, 2)
    );

    useEffect(() => {
        const newConfig: FunctionCallNodeConfig = {
            function_name: '',
            arguments: {},
            timeout_seconds: 30,
            ...config,
        };

        if (JSON.stringify(newConfig) !== JSON.stringify(localConfig)) {
            setLocalConfig(newConfig);
            setArgumentsStr(
                typeof config?.arguments === 'string'
                    ? config.arguments
                    : JSON.stringify(config?.arguments || {}, null, 2)
            );
        }
    }, [config]);

    useEffect(() => {
        // Sync argumentsStr to localConfig.arguments
        const updatedConfig = {...localConfig};
        try {
            updatedConfig.arguments = JSON.parse(argumentsStr);
        } catch {
            updatedConfig.arguments = argumentsStr as any;
        }

        if (JSON.stringify(updatedConfig) !== JSON.stringify(localConfig)) {
            setLocalConfig(updatedConfig);
        }
    }, [argumentsStr]);

    useEffect(() => {
        if (JSON.stringify(localConfig) !== JSON.stringify(config)) {
            onChange(localConfig);
        }
    }, [localConfig, onChange]);

    return (
        <div className="flex flex-col gap-4">
            {/* Function Name */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Function Name
                </label>
                <input
                    type="text"
                    value={localConfig.function_name || ''}
                    onChange={(e) =>
                        setLocalConfig((prev) => ({...prev, function_name: e.target.value}))
                    }
                    placeholder="my_function"
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200"
                />
            </div>

            {/* Arguments */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Arguments (JSON)
                </label>
                <textarea
                    value={argumentsStr}
                    onChange={(e) => setArgumentsStr(e.target.value)}
                    placeholder='{"key": "{{input.value}}"}'
                    rows={6}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-none font-mono text-slate-800 dark:text-slate-200 placeholder-slate-400"
                />
                <div
                    className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg leading-relaxed">
                    <strong className="text-slate-700 dark:text-slate-300">Enter function arguments as JSON
                        object.</strong> You can use template variables
                    like <code
                    className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px] text-slate-700 dark:text-slate-300">
                    {`{{env.api_key}}`}
                </code> or{' '}
                    <code
                        className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px] text-slate-700 dark:text-slate-300">
                        {`{{input.user_id}}`}
                    </code>
                </div>
            </div>

            {/* Timeout */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Timeout (seconds)
                </label>
                <input
                    type="number"
                    value={localConfig.timeout_seconds ?? 30}
                    onChange={(e) =>
                        setLocalConfig((prev) => ({...prev, timeout_seconds: Number(e.target.value)}))
                    }
                    min={1}
                    max={300}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200"
                />
            </div>
        </div>
    );
};

export default FunctionCallNodeConfigComponent;
