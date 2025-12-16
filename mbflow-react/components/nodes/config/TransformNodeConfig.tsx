import React, {useEffect, useState} from 'react';
import type {TransformNodeConfig} from '@/types/nodeConfigs.ts';
import {TRANSFORM_LANGUAGES} from '@/types/nodeConfigs.ts';

interface TransformNodeConfigProps {
    config: TransformNodeConfig;
    nodeId?: string;
    onChange: (config: TransformNodeConfig) => void;
}

export const TransformNodeConfigComponent: React.FC<TransformNodeConfigProps> = ({
                                                                                     config,
                                                                                     nodeId,
                                                                                     onChange,
                                                                                 }) => {
    const [localConfig, setLocalConfig] = useState<TransformNodeConfig>({...config});

    useEffect(() => {
        if (JSON.stringify(config) !== JSON.stringify(localConfig)) {
            setLocalConfig({...config});
        }
    }, [config]);

    useEffect(() => {
        if (JSON.stringify(localConfig) !== JSON.stringify(config)) {
            onChange(localConfig);
        }
    }, [localConfig]);

    const handleLanguageChange = (language: 'jq' | 'javascript') => {
        setLocalConfig((prev) => ({...prev, language}));
    };

    const handleExpressionChange = (expression: string) => {
        setLocalConfig((prev) => ({...prev, expression}));
    };

    const handleTimeoutChange = (timeout_seconds: number) => {
        setLocalConfig((prev) => ({...prev, timeout_seconds}));
    };

    const getPlaceholder = () => {
        return localConfig.language === 'jq'
            ? '.field | select(.value > 0)'
            : 'return input.field * 2;';
    };

    const getHintText = () => {
        return localConfig.language === 'jq' ? 'jq filter' : 'JavaScript function';
    };

    return (
        <div className="flex flex-col gap-4">
            {/* Language Selection */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Language
                </label>
                <select
                    value={localConfig.language}
                    onChange={(e) => handleLanguageChange(e.target.value as 'jq' | 'javascript')}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200"
                >
                    {TRANSFORM_LANGUAGES.map((lang) => (
                        <option key={lang} value={lang}>
                            {lang.toUpperCase()}
                        </option>
                    ))}
                </select>
            </div>

            {/* Expression */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400 flex items-center gap-2">
                    Expression
                    <span className="text-[11px] font-normal text-slate-400">
            {getHintText()}
          </span>
                </label>
                <textarea
                    value={localConfig.expression}
                    onChange={(e) => handleExpressionChange(e.target.value)}
                    placeholder={getPlaceholder()}
                    rows={10}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-none font-mono text-slate-800 dark:text-slate-200 placeholder-slate-400"
                />

                {/* Help Text */}
                <div
                    className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg leading-relaxed">
                    {localConfig.language === 'jq' ? (
                        <>
                            <strong className="text-slate-700 dark:text-slate-300">jq Examples:</strong>
                            <br/>
                            <code
                                className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">
                                .
                            </code>{' '}
                            - Pass through input
                            <br/>
                            <code
                                className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">
                                .field
                            </code>{' '}
                            - Extract field
                            <br/>
                            <code
                                className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">
                                {`{ name: .user.name, count: .items | length }`}
                            </code>{' '}
                            - Transform structure
                        </>
                    ) : (
                        <>
                            <strong className="text-slate-700 dark:text-slate-300">JavaScript Example:</strong>
                            <br/>
                            <code
                                className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">
                                {`return { name: input.user.name, count: input.items.length };`}
                            </code>
                        </>
                    )}
                </div>
            </div>

            {/* Timeout */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Timeout (seconds)
                </label>
                <input
                    type="number"
                    value={localConfig.timeout_seconds ?? 10}
                    onChange={(e) => handleTimeoutChange(Number(e.target.value))}
                    min={1}
                    max={60}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200"
                />
            </div>
        </div>
    );
};
