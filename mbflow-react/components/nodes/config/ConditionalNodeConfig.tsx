/**
 * ConditionalNodeConfig - React component for configuring conditional nodes
 *
 * Ported from: /mbflow-ui/src/components/nodes/config/ConditionalNodeConfig.vue
 *
 * Features:
 * - Condition expression editor with template variable support
 * - Expression examples with different use cases
 * - Important notes about branch execution
 * - Supported operators reference
 *
 * Usage:
 * ```tsx
 * <ConditionalNodeConfig
 *   config={conditionalConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => console.log(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from 'react';
import {ConditionalNodeConfig} from '@/types/nodeConfigs.ts';
import {VariableAutocomplete} from '@/components';
import { useTranslation } from '@/store/translations';

interface ConditionalNodeConfigProps {
    config: ConditionalNodeConfig;
    nodeId?: string;
    onChange: (config: ConditionalNodeConfig) => void;
}

export const ConditionalNodeConfigComponent: React.FC<ConditionalNodeConfigProps> = ({
                                                                                         config,
                                                                                         nodeId,
                                                                                         onChange,
                                                                                     }) => {
    const t = useTranslation();
    const [localConfig, setLocalConfig] = useState<ConditionalNodeConfig>({
        ...config,
        condition: config.condition || '{{input.value}} > 0',
    });

    useEffect(() => {
        const newConfig = {
            ...config,
            condition: config.condition || '{{input.value}} > 0',
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

    const handleConditionChange = (condition: string) => {
        setLocalConfig((prev) => ({...prev, condition}));
    };

    return (
        <div className="flex flex-col gap-4">
            {/* Condition Expression */}
            <div className="flex flex-col gap-1.5">
                <label className="text-[13px] font-semibold text-slate-700 dark:text-slate-300">
                    {t.nodeConfig.conditional.conditionExpression}
                </label>
                <VariableAutocomplete
                    value={localConfig.condition || ''}
                    onChange={handleConditionChange}
                    placeholder={t.nodeConfig.conditional.conditionPlaceholder}
                    type="textarea"
                    rows={4}
                    className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg text-sm font-mono bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent resize-vertical min-h-[80px]"
                />
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {t.nodeConfig.conditional.conditionHint}
                </p>
            </div>

            {/* Expression Examples */}
            <div className="p-4 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                <h4 className="text-[13px] font-bold text-blue-900 dark:text-blue-100 mb-3">
                    💡 {t.nodeConfig.conditional.examples.title}
                </h4>
                <ul className="flex flex-col gap-2 pl-5 list-disc">
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        <code
                            className="bg-white dark:bg-slate-800 px-1.5 py-0.5 rounded border border-blue-100 dark:border-blue-900 font-mono text-[11px] text-blue-700 dark:text-blue-300">
                            {'{{input.value}} > 100'}
                        </code>
                        {' - ' + t.nodeConfig.conditional.examples.numeric.split(' - ')[1]}
                    </li>
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        <code
                            className="bg-white dark:bg-slate-800 px-1.5 py-0.5 rounded border border-blue-100 dark:border-blue-900 font-mono text-[11px] text-blue-700 dark:text-blue-300">
                            {'{{input.status}} == "active"'}
                        </code>
                        {' - ' + t.nodeConfig.conditional.examples.string.split(' - ')[1]}
                    </li>
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        <code
                            className="bg-white dark:bg-slate-800 px-1.5 py-0.5 rounded border border-blue-100 dark:border-blue-900 font-mono text-[11px] text-blue-700 dark:text-blue-300">
                            {'{{input.count}} > 0 && {{input.enabled}}'}
                        </code>
                        {' - ' + t.nodeConfig.conditional.examples.multiple.split(' - ')[1]}
                    </li>
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        <code
                            className="bg-white dark:bg-slate-800 px-1.5 py-0.5 rounded border border-blue-100 dark:border-blue-900 font-mono text-[11px] text-blue-700 dark:text-blue-300">
                            {'len({{input.items}}) > 5'}
                        </code>
                        {' - ' + t.nodeConfig.conditional.examples.arrayLength.split(' - ')[1]}
                    </li>
                </ul>
            </div>

            {/* Important Notes */}
            <div
                className="p-4 bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                <h4 className="text-[13px] font-bold text-amber-900 dark:text-amber-100 mb-3">
                    ⚠️ {t.nodeConfig.conditional.notes.title}
                </h4>
                <ul className="flex flex-col gap-2 pl-5 list-disc">
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        {t.nodeConfig.conditional.notes.trueBranch}
                    </li>
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        {t.nodeConfig.conditional.notes.falseBranch}
                    </li>
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        {t.nodeConfig.conditional.notes.execution}
                    </li>
                    <li className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        {t.nodeConfig.conditional.notes.templates}
                    </li>
                </ul>
            </div>

            {/* Supported Operators */}
            <div className="p-4 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg">
                <h4 className="text-[13px] font-bold text-slate-700 dark:text-slate-300 mb-3">
                    📚 {t.nodeConfig.conditional.operators.title}
                </h4>
                <div className="grid grid-cols-2 gap-3">
                    <div>
                        <h5 className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase mb-1">
                            {t.nodeConfig.conditional.operators.comparison}
                        </h5>
                        <code
                            className="block text-xs bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700 font-mono text-slate-700 dark:text-slate-300">
                            {t.nodeConfig.conditional.operators.comparisonOps}
                        </code>
                    </div>
                    <div>
                        <h5 className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase mb-1">
                            {t.nodeConfig.conditional.operators.logical}
                        </h5>
                        <code
                            className="block text-xs bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700 font-mono text-slate-700 dark:text-slate-300">
                            {t.nodeConfig.conditional.operators.logicalOps}
                        </code>
                    </div>
                    <div>
                        <h5 className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase mb-1">
                            {t.nodeConfig.conditional.operators.arithmetic}
                        </h5>
                        <code
                            className="block text-xs bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700 font-mono text-slate-700 dark:text-slate-300">
                            {t.nodeConfig.conditional.operators.arithmeticOps}
                        </code>
                    </div>
                    <div>
                        <h5 className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase mb-1">
                            {t.nodeConfig.conditional.operators.functions}
                        </h5>
                        <code
                            className="block text-xs bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700 font-mono text-slate-700 dark:text-slate-300">
                            {t.nodeConfig.conditional.operators.functionsOps}
                        </code>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ConditionalNodeConfigComponent;
