import React from 'react';
import { Zap, ArrowRight, FileText, Code } from 'lucide-react';
import type {TransformNodeConfig} from '@/types/nodeConfigs';
import {TRANSFORM_TYPES} from '@/types/nodeConfigs';
import {VariableAutocomplete} from '../../builder/VariableAutocomplete';

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
    // Create safeConfig with defaults to prevent undefined errors
    const safeConfig: TransformNodeConfig = {
        type: config?.type || 'passthrough',
        template: config?.template || '',
        expression: config?.expression || '',
        filter: config?.filter || '.',
        timeout_seconds: config?.timeout_seconds ?? 10,
    };

    // Handlers call onChange directly with safeConfig spread
    const handleTypeChange = (type: TransformNodeConfig['type']) => {
        onChange({...safeConfig, type});
    };

    const handleTemplateChange = (template: string) => {
        onChange({...safeConfig, template});
    };

    const handleExpressionChange = (expression: string) => {
        onChange({...safeConfig, expression});
    };

    const handleFilterChange = (filter: string) => {
        onChange({...safeConfig, filter});
    };

    const handleTimeoutChange = (timeout_seconds: number) => {
        onChange({...safeConfig, timeout_seconds});
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-900/10 dark:to-orange-900/10 border border-amber-200 dark:border-amber-800 rounded-lg p-4 flex items-start gap-3">
                <Zap className="text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" size={18}/>
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">Transform Data</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        Transform input data using various methods: passthrough, templates, expressions, or jq filters
                    </p>
                </div>
            </div>

            {/* Transformation Type */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Transformation Type
                    </span>
                    <select
                        value={safeConfig.type}
                        onChange={(e) => handleTypeChange(e.target.value as TransformNodeConfig['type'])}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 text-sm"
                    >
                        {TRANSFORM_TYPES.map((type) => (
                            <option key={type} value={type}>
                                {type.charAt(0).toUpperCase() + type.slice(1)}
                            </option>
                        ))}
                    </select>
                </label>

                {/* Type description */}
                <div className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg">
                    {safeConfig.type === 'passthrough' && (
                        <div className="flex items-start gap-2">
                            <ArrowRight size={14} className="text-amber-500 flex-shrink-0 mt-0.5" />
                            <div>
                                <strong className="text-slate-700 dark:text-slate-300">Passthrough:</strong> Pass input data unchanged to the next node
                            </div>
                        </div>
                    )}
                    {safeConfig.type === 'template' && (
                        <div className="flex items-start gap-2">
                            <FileText size={14} className="text-amber-500 flex-shrink-0 mt-0.5" />
                            <div>
                                <strong className="text-slate-700 dark:text-slate-300">Template:</strong> Use template syntax with variables like <code className="bg-slate-200 dark:bg-slate-800 px-1 py-0.5 rounded text-[11px]">{'{{env.var}}'}</code> or <code className="bg-slate-200 dark:bg-slate-800 px-1 py-0.5 rounded text-[11px]">{'{{input.field}}'}</code>
                            </div>
                        </div>
                    )}
                    {safeConfig.type === 'expression' && (
                        <div className="flex items-start gap-2">
                            <Code size={14} className="text-amber-500 flex-shrink-0 mt-0.5" />
                            <div>
                                <strong className="text-slate-700 dark:text-slate-300">Expression:</strong> Use expr-lang for complex transformations with access to <code className="bg-slate-200 dark:bg-slate-800 px-1 py-0.5 rounded text-[11px]">input</code> variable
                            </div>
                        </div>
                    )}
                    {safeConfig.type === 'jq' && (
                        <div className="flex items-start gap-2">
                            <Code size={14} className="text-amber-500 flex-shrink-0 mt-0.5" />
                            <div>
                                <strong className="text-slate-700 dark:text-slate-300">JQ Filter:</strong> Use jq syntax for JSON transformations and queries
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Template field (only for type: template) */}
            {safeConfig.type === 'template' && (
                <div className="space-y-3">
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            Template String
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.template}
                            onChange={handleTemplateChange}
                            placeholder="Hello {{env.userName}}, your input is: {{input.data}}"
                            nodeId={nodeId}
                            rows={6}
                        />
                    </label>
                    <div className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg leading-relaxed">
                        <strong className="text-slate-700 dark:text-slate-300">Template Examples:</strong>
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">{'{{env.apiKey}}'}</code> - Access workflow/execution variable
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">{'{{input.user.name}}'}</code> - Access parent node output
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">{'{{input.items[0].id}}'}</code> - Array access
                    </div>
                </div>
            )}

            {/* Expression field (only for type: expression) */}
            {safeConfig.type === 'expression' && (
                <div className="space-y-3">
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            Expression (expr-lang)
                        </span>
                        <textarea
                            value={safeConfig.expression}
                            onChange={(e) => handleExpressionChange(e.target.value)}
                            placeholder='input.field * 2'
                            rows={8}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 text-sm font-mono resize-none"
                        />
                    </label>
                    <div className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg leading-relaxed">
                        <strong className="text-slate-700 dark:text-slate-300">Expression Examples:</strong>
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">input.value * 100</code> - Math operations
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">input.name + " suffix"</code> - String concatenation
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">{'filter(input.items, {# > 10})'}</code> - Array filtering
                    </div>
                </div>
            )}

            {/* JQ Filter field (only for type: jq) */}
            {safeConfig.type === 'jq' && (
                <div className="space-y-3">
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            JQ Filter
                        </span>
                        <textarea
                            value={safeConfig.filter}
                            onChange={(e) => handleFilterChange(e.target.value)}
                            placeholder='.field | select(.value > 0)'
                            rows={8}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 text-sm font-mono resize-none"
                        />
                    </label>
                    <div className="text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg leading-relaxed">
                        <strong className="text-slate-700 dark:text-slate-300">JQ Examples:</strong>
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">.</code> - Pass through input
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">.field</code> - Extract field
                        <br/>
                        <code className="bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded font-mono text-[11px]">{'{ name: .user.name, count: .items | length }'}</code> - Transform structure
                    </div>
                </div>
            )}

            {/* Timeout */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Timeout (seconds)
                    </span>
                    <input
                        type="number"
                        value={safeConfig.timeout_seconds}
                        onChange={(e) => handleTimeoutChange(Number(e.target.value))}
                        min={1}
                        max={60}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 text-sm"
                    />
                </label>
            </div>
        </div>
    );
};
