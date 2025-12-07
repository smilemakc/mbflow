import React, {useEffect, useState} from 'react';
import {MergeNodeConfig} from '@/types/nodeConfigs.ts';
import {VariableAutocomplete} from '@/components';

interface Props {
    config: MergeNodeConfig;
    nodeId?: string;
    onChange: (config: MergeNodeConfig) => void;
}

export const MergeNodeConfigComponent: React.FC<Props> = ({config, nodeId, onChange}) => {
    const [localConfig, setLocalConfig] = useState<MergeNodeConfig>({
        merge_strategy: 'all',
        ...config,
    });

    useEffect(() => {
        if (JSON.stringify(config) !== JSON.stringify(localConfig)) {
            setLocalConfig({...config});
        }
    }, [config]);

    useEffect(() => {
        onChange(localConfig);
    }, [localConfig]);

    const handleStrategyChange = (strategy: 'first' | 'last' | 'all' | 'custom') => {
        setLocalConfig({...localConfig, merge_strategy: strategy});
    };

    const handleExpressionChange = (expression: string) => {
        setLocalConfig({...localConfig, custom_expression: expression});
    };

    return (
        <div className="flex flex-col gap-4">
            {/* Merge Strategy */}
            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Merge Strategy
                </label>
                <select
                    value={localConfig.merge_strategy || 'all'}
                    onChange={(e) => handleStrategyChange(e.target.value as any)}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200"
                >
                    <option value="first">First (Use first available result)</option>
                    <option value="last">Last (Use last available result)</option>
                    <option value="all">All (Combine all results into array)</option>
                    <option value="custom">Custom (Use expression)</option>
                </select>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Choose how to combine results from multiple parent nodes.
                </p>
            </div>

            {/* Custom Expression (only when strategy is 'custom') */}
            {localConfig.merge_strategy === 'custom' && (
                <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                        Custom Merge Expression
                    </label>
                    <VariableAutocomplete
                        type="textarea"
                        value={localConfig.custom_expression || ''}
                        onChange={handleExpressionChange}
                        rows={4}
                        placeholder="[{{input.parent1}}, {{input.parent2}}]"
                        className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-none text-slate-800 dark:text-slate-200 placeholder-slate-400 font-mono"
                    />
                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        Use expr-lang to define custom merge logic.
                    </p>
                </div>
            )}

            {/* Strategy Examples */}
            <div
                className="p-4 bg-purple-50 dark:bg-purple-950/20 border border-purple-200 dark:border-purple-900/30 rounded-lg">
                <h4 className="text-xs font-bold text-purple-800 dark:text-purple-400 mb-3">
                    💡 Merge Strategy Examples
                </h4>
                <div className="flex flex-col gap-3">
                    <div
                        className="p-3 bg-white dark:bg-slate-900 border border-purple-100 dark:border-purple-900/50 rounded-md">
                        <div className="text-xs font-semibold text-purple-700 dark:text-purple-300 mb-1">
                            📥 First
                        </div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-2 leading-relaxed">
                            Returns the output of the first parent node that completes.
                        </div>
                        <code
                            className="block bg-purple-50 dark:bg-purple-950/30 px-2.5 py-1.5 rounded text-[11px] text-purple-700 dark:text-purple-300 border border-purple-100 dark:border-purple-900/50 font-mono">
                            Result: parent1_output
                        </code>
                    </div>

                    <div
                        className="p-3 bg-white dark:bg-slate-900 border border-purple-100 dark:border-purple-900/50 rounded-md">
                        <div className="text-xs font-semibold text-purple-700 dark:text-purple-300 mb-1">
                            📤 Last
                        </div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-2 leading-relaxed">
                            Returns the output of the last parent node that completes.
                        </div>
                        <code
                            className="block bg-purple-50 dark:bg-purple-950/30 px-2.5 py-1.5 rounded text-[11px] text-purple-700 dark:text-purple-300 border border-purple-100 dark:border-purple-900/50 font-mono">
                            Result: parent3_output
                        </code>
                    </div>

                    <div
                        className="p-3 bg-white dark:bg-slate-900 border border-purple-100 dark:border-purple-900/50 rounded-md">
                        <div className="text-xs font-semibold text-purple-700 dark:text-purple-300 mb-1">
                            📦 All
                        </div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-2 leading-relaxed">
                            Combines all parent outputs into an array.
                        </div>
                        <code
                            className="block bg-purple-50 dark:bg-purple-950/30 px-2.5 py-1.5 rounded text-[11px] text-purple-700 dark:text-purple-300 border border-purple-100 dark:border-purple-900/50 font-mono">
                            Result: [parent1, parent2, parent3]
                        </code>
                    </div>

                    <div
                        className="p-3 bg-white dark:bg-slate-900 border border-purple-100 dark:border-purple-900/50 rounded-md">
                        <div className="text-xs font-semibold text-purple-700 dark:text-purple-300 mb-1">
                            ⚙️ Custom
                        </div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-2 leading-relaxed">
                            Use custom expression to merge outputs.
                        </div>
                        <code
                            className="block bg-purple-50 dark:bg-purple-950/30 px-2.5 py-1.5 rounded text-[11px] text-purple-700 dark:text-purple-300 border border-purple-100 dark:border-purple-900/50 font-mono">
                            Result: your_expression
                        </code>
                    </div>
                </div>
            </div>

            {/* How It Works */}
            <div
                className="p-4 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900/30 rounded-lg">
                <h4 className="text-xs font-bold text-blue-800 dark:text-blue-400 mb-3">
                    ℹ️ How It Works
                </h4>
                <ul className="space-y-2 pl-5 list-disc">
                    <li className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                        Merge node waits for <strong>all parent nodes</strong> to complete
                    </li>
                    <li className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                        Parent outputs are accessible via{' '}
                        <code
                            className="bg-white dark:bg-slate-900 px-1.5 py-0.5 rounded text-[11px] text-purple-700 dark:text-purple-300 border border-purple-100 dark:border-purple-900/50 font-mono">
                            {'{{input.parentNodeId}}'}
                        </code>
                    </li>
                    <li className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                        For "all" strategy, outputs are combined into a single array
                    </li>
                    <li className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                        Use "custom" strategy for complex merging logic (filtering, transforming, etc.)
                    </li>
                </ul>
            </div>

            {/* Custom Expression Examples */}
            <div
                className="p-4 bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-800 rounded-lg">
                <h4 className="text-xs font-bold text-slate-700 dark:text-slate-300 mb-3">
                    📚 Custom Expression Examples
                </h4>
                <div className="flex flex-col gap-3">
                    <div
                        className="p-3 bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-md">
                        <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1.5">
                            Merge objects:
                        </div>
                        <code
                            className="block text-xs whitespace-pre-wrap break-all font-mono text-slate-700 dark:text-slate-300">
                            {'{"a": {{input.node1}}.value, "b": {{input.node2}}.value}'}
                        </code>
                    </div>

                    <div
                        className="p-3 bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-md">
                        <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1.5">
                            Filter and combine:
                        </div>
                        <code
                            className="block text-xs whitespace-pre-wrap break-all font-mono text-slate-700 dark:text-slate-300">
                            {'filter({{input.results}}, {.status == "success"})'}
                        </code>
                    </div>

                    <div
                        className="p-3 bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-md">
                        <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1.5">
                            Sum values:
                        </div>
                        <code
                            className="block text-xs whitespace-pre-wrap break-all font-mono text-slate-700 dark:text-slate-300">
                            {'{{input.node1}}.count + {{input.node2}}.count'}
                        </code>
                    </div>
                </div>
            </div>
        </div>
    );
};
