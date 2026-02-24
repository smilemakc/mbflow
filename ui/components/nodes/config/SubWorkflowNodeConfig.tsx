import React, {useEffect, useState} from 'react';
import {SubWorkflowNodeConfig} from '@/types/nodeConfigs';
import {useTranslation} from '@/store/translations';

interface SubWorkflowNodeConfigProps {
    config: SubWorkflowNodeConfig;
    nodeId?: string;
    onChange: (config: SubWorkflowNodeConfig) => void;
}

export const SubWorkflowNodeConfigComponent: React.FC<SubWorkflowNodeConfigProps> = ({
    config,
    onChange,
}) => {
    const t = useTranslation();
    const tc = (t.nodeConfig as any).subWorkflow;

    const [localConfig, setLocalConfig] = useState<SubWorkflowNodeConfig>({
        workflow_id: config.workflow_id || '',
        for_each: config.for_each || 'input.items',
        item_var: config.item_var || 'item',
        max_parallelism: config.max_parallelism ?? 0,
        on_error: config.on_error || 'fail_fast',
    });

    useEffect(() => {
        const newConfig: SubWorkflowNodeConfig = {
            workflow_id: config.workflow_id || '',
            for_each: config.for_each || 'input.items',
            item_var: config.item_var || 'item',
            max_parallelism: config.max_parallelism ?? 0,
            on_error: config.on_error || 'fail_fast',
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

    const handleChange = (field: keyof SubWorkflowNodeConfig, value: any) => {
        setLocalConfig(prev => ({...prev, [field]: value}));
    };

    return (
        <div className="flex flex-col gap-4">
            {/* Workflow ID */}
            <div className="flex flex-col gap-1.5">
                <label className="text-[13px] font-semibold text-slate-700 dark:text-slate-300">
                    {tc.workflowId}
                </label>
                <input
                    type="text"
                    value={localConfig.workflow_id}
                    onChange={e => handleChange('workflow_id', e.target.value)}
                    placeholder={tc.workflowIdPlaceholder}
                    className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg text-sm font-mono bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-indigo-400 focus:border-transparent"
                />
                <p className="text-xs text-slate-500 dark:text-slate-400">
                    {tc.workflowIdHint}
                </p>
            </div>

            {/* For Each Expression */}
            <div className="flex flex-col gap-1.5">
                <label className="text-[13px] font-semibold text-slate-700 dark:text-slate-300">
                    {tc.forEach}
                </label>
                <input
                    type="text"
                    value={localConfig.for_each}
                    onChange={e => handleChange('for_each', e.target.value)}
                    placeholder={tc.forEachPlaceholder}
                    className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg text-sm font-mono bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-indigo-400 focus:border-transparent"
                />
                <p className="text-xs text-slate-500 dark:text-slate-400">
                    {tc.forEachHint}
                </p>
            </div>

            {/* Item Variable Name */}
            <div className="flex flex-col gap-1.5">
                <label className="text-[13px] font-semibold text-slate-700 dark:text-slate-300">
                    {tc.itemVar}
                </label>
                <input
                    type="text"
                    value={localConfig.item_var || 'item'}
                    onChange={e => handleChange('item_var', e.target.value)}
                    placeholder={tc.itemVarPlaceholder}
                    className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg text-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-indigo-400 focus:border-transparent"
                />
                <p className="text-xs text-slate-500 dark:text-slate-400">
                    {tc.itemVarHint}
                </p>
            </div>

            {/* Max Parallelism */}
            <div className="flex flex-col gap-1.5">
                <label className="text-[13px] font-semibold text-slate-700 dark:text-slate-300">
                    {tc.maxParallelism}
                </label>
                <input
                    type="number"
                    min={0}
                    value={localConfig.max_parallelism ?? 0}
                    onChange={e => handleChange('max_parallelism', parseInt(e.target.value) || 0)}
                    className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg text-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-indigo-400 focus:border-transparent"
                />
                <p className="text-xs text-slate-500 dark:text-slate-400">
                    {tc.maxParallelismHint}
                </p>
            </div>

            {/* Error Handling */}
            <div className="flex flex-col gap-1.5">
                <label className="text-[13px] font-semibold text-slate-700 dark:text-slate-300">
                    {tc.onError}
                </label>
                <select
                    value={localConfig.on_error || 'fail_fast'}
                    onChange={e => handleChange('on_error', e.target.value)}
                    className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg text-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-indigo-400 focus:border-transparent"
                >
                    <option value="fail_fast">{tc.onErrorFailFast} — {tc.onErrorFailFastHint}</option>
                    <option value="collect_partial">{tc.onErrorCollectPartial} — {tc.onErrorCollectPartialHint}</option>
                </select>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                    {tc.onErrorHint}
                </p>
            </div>

            {/* How It Works */}
            <div className="p-4 bg-indigo-50 dark:bg-indigo-950/20 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                <h4 className="text-[13px] font-bold text-indigo-900 dark:text-indigo-100 mb-3">
                    {tc.howItWorks}
                </h4>
                <ol className="flex flex-col gap-2 pl-5 list-decimal">
                    {(tc.howItWorksItems as string[]).map((item: string, i: number) => (
                        <li key={i} className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                            {item}
                        </li>
                    ))}
                </ol>
            </div>
        </div>
    );
};

export default SubWorkflowNodeConfigComponent;
