import React, {useState} from 'react';
import {
    ArrowRight,
    Braces,
    Check,
    ChevronDown,
    ChevronRight,
    Code2,
    Copy,
    FileInput,
    Globe,
    Info,
    Server,
    Variable,
    X
} from 'lucide-react';
import {Button} from '../ui';

interface VariableCategory {
    id: string;
    title: string;
    description: string;
    icon: React.ReactNode;
    examples: {
        syntax: string;
        description: string;
    }[];
}

const VARIABLE_CATEGORIES: VariableCategory[] = [
    {
        id: 'env',
        title: 'Environment Variables (env)',
        description: 'Access workflow-level and execution-level variables. Execution variables override workflow variables.',
        icon: <Globe size={18} className="text-green-500"/>,
        examples: [
            {syntax: '{{env.API_KEY}}', description: 'Access API key from workflow variables'},
            {syntax: '{{env.BASE_URL}}', description: 'Access base URL configuration'},
            {syntax: '{{env.TELEGRAM_BOT_TOKEN}}', description: 'Access Telegram bot token'},
            {syntax: '{{env.DEBUG_MODE}}', description: 'Access debug mode flag'},
        ]
    },
    {
        id: 'input',
        title: 'Input Variables (input)',
        description: 'Access data passed from parent nodes. The input object contains merged outputs from all connected parent nodes.',
        icon: <FileInput size={18} className="text-blue-500"/>,
        examples: [
            {syntax: '{{input.message}}', description: 'Access message from parent node output'},
            {syntax: '{{input.user.name}}', description: 'Access nested user name field'},
            {syntax: '{{input.items[0].id}}', description: 'Access first item ID from array'},
            {syntax: '{{input.response.data.results}}', description: 'Access deeply nested response data'},
        ]
    },
    {
        id: 'node',
        title: 'Node Outputs (Direct Reference)',
        description: 'Reference output from specific nodes by their ID. Useful when you need data from a non-parent node.',
        icon: <Server size={18} className="text-purple-500"/>,
        examples: [
            {syntax: '{{http.body}}', description: 'Access HTTP node response body'},
            {syntax: '{{llm.content}}', description: 'Access LLM generated content'},
            {syntax: '{{transform.result}}', description: 'Access transform node result'},
            {syntax: '{{http_2.headers.content-type}}', description: 'Access second HTTP node headers'},
        ]
    }
];

const TRANSFORM_MODES = [
    {
        name: 'Passthrough',
        description: 'Pass input data unchanged to output',
        example: '// No transformation needed\n// Input flows directly to output'
    },
    {
        name: 'Template',
        description: 'String substitution with template syntax',
        example: 'Hello, {{input.user.name}}!\nYour order #{{input.orderId}} is ready.'
    },
    {
        name: 'Expression (expr-lang)',
        description: 'Complex transformations using expr-lang',
        example: 'input.items.filter(i, i.price > 100).map(i, i.name)'
    },
    {
        name: 'JQ',
        description: 'JSON query and transformation with jq syntax',
        example: '.items | map(select(.active)) | sort_by(.name)'
    }
];

interface WorkflowVariablesGuideProps {
    isModal?: boolean;
    onClose?: () => void;
}

export const WorkflowVariablesGuide: React.FC<WorkflowVariablesGuideProps> = ({isModal = false, onClose}) => {
    const [expandedCategories, setExpandedCategories] = useState<string[]>(['env', 'input']);
    const [copiedSyntax, setCopiedSyntax] = useState<string | null>(null);
    const [showTransformModes, setShowTransformModes] = useState(false);

    const toggleCategory = (categoryId: string) => {
        setExpandedCategories(prev =>
            prev.includes(categoryId)
                ? prev.filter(id => id !== categoryId)
                : [...prev, categoryId]
        );
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        setCopiedSyntax(text);
        setTimeout(() => setCopiedSyntax(null), 2000);
    };

    const content = (
        <div
            className={`bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden ${isModal ? 'max-h-[80vh] overflow-y-auto' : ''}`}>
            {/* Header */}
            <div
                className="p-5 border-b border-slate-100 dark:border-slate-800 sticky top-0 bg-white dark:bg-slate-900 z-10">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg">
                            <Variable size={20} className="text-indigo-600 dark:text-indigo-400"/>
                        </div>
                        <div>
                            <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                                Workflow Variables & Templates
                            </h2>
                            <p className="text-sm text-slate-500 dark:text-slate-400">
                                Reference guide for using variables in your workflows
                            </p>
                        </div>
                    </div>
                    {isModal && onClose && (
                        <Button
                            variant="ghost"
                            size="sm"
                            icon={<X size={20}/>}
                            onClick={onClose}
                            title="Close"
                        />
                    )}
                </div>
            </div>

            {/* Info Banner */}
            <div
                className="mx-5 mt-5 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/30 rounded-lg">
                <div className="flex gap-3">
                    <Info size={18} className="text-blue-600 dark:text-blue-400 shrink-0 mt-0.5"/>
                    <div className="text-sm text-blue-800 dark:text-blue-300">
                        <p className="font-medium mb-1">Template Syntax</p>
                        <p className="text-blue-700 dark:text-blue-400">
                            Use <code
                            className="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/40 rounded text-xs font-mono">{'{{variable.path}}'}</code> syntax
                            to reference variables in node configurations. Templates are resolved before node execution.
                        </p>
                    </div>
                </div>
            </div>

            {/* Variable Categories */}
            <div className="p-5 space-y-3">
                {VARIABLE_CATEGORIES.map(category => (
                    <div
                        key={category.id}
                        className="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden"
                    >
                        {/* Category Header */}
                        <button
                            onClick={() => toggleCategory(category.id)}
                            className="w-full flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                        >
                            <div className="flex items-center gap-3">
                                {category.icon}
                                <div className="text-left">
                                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">
                                        {category.title}
                                    </h3>
                                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                        {category.description}
                                    </p>
                                </div>
                            </div>
                            {expandedCategories.includes(category.id) ? (
                                <ChevronDown size={18} className="text-slate-400"/>
                            ) : (
                                <ChevronRight size={18} className="text-slate-400"/>
                            )}
                        </button>

                        {/* Category Examples */}
                        {expandedCategories.includes(category.id) && (
                            <div className="p-4 space-y-2 bg-white dark:bg-slate-900">
                                {category.examples.map((example, idx) => (
                                    <div
                                        key={idx}
                                        className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg group"
                                    >
                                        <div className="flex items-center gap-3 min-w-0">
                                            <code
                                                className="px-2 py-1 bg-slate-200 dark:bg-slate-700 rounded text-xs font-mono text-slate-800 dark:text-slate-200 shrink-0">
                                                {example.syntax}
                                            </code>
                                            <ArrowRight size={14} className="text-slate-400 shrink-0"/>
                                            <span className="text-sm text-slate-600 dark:text-slate-400 truncate">
                        {example.description}
                      </span>
                                        </div>
                                        <button
                                            onClick={() => copyToClipboard(example.syntax)}
                                            className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity"
                                            title="Copy to clipboard"
                                        >
                                            {copiedSyntax === example.syntax ? (
                                                <Check size={14} className="text-green-500"/>
                                            ) : (
                                                <Copy size={14}/>
                                            )}
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                ))}
            </div>

            {/* Transform Modes Section */}
            <div className="px-5 pb-5">
                <button
                    onClick={() => setShowTransformModes(!showTransformModes)}
                    className="w-full flex items-center justify-between p-4 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-900/20 dark:to-indigo-900/20 border border-purple-200 dark:border-purple-900/30 rounded-lg hover:from-purple-100 hover:to-indigo-100 dark:hover:from-purple-900/30 dark:hover:to-indigo-900/30 transition-colors"
                >
                    <div className="flex items-center gap-3">
                        <Braces size={18} className="text-purple-600 dark:text-purple-400"/>
                        <div className="text-left">
                            <h3 className="font-semibold text-purple-900 dark:text-purple-300 text-sm">
                                Transform Modes
                            </h3>
                            <p className="text-xs text-purple-700 dark:text-purple-400 mt-0.5">
                                Different ways to transform data in Transform nodes
                            </p>
                        </div>
                    </div>
                    {showTransformModes ? (
                        <ChevronDown size={18} className="text-purple-400"/>
                    ) : (
                        <ChevronRight size={18} className="text-purple-400"/>
                    )}
                </button>

                {showTransformModes && (
                    <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-3">
                        {TRANSFORM_MODES.map((mode, idx) => (
                            <div
                                key={idx}
                                className="p-4 bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg"
                            >
                                <h4 className="font-medium text-slate-900 dark:text-white text-sm mb-1">
                                    {mode.name}
                                </h4>
                                <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                                    {mode.description}
                                </p>
                                <pre
                                    className="p-3 bg-slate-900 dark:bg-slate-950 rounded text-xs font-mono text-green-400 overflow-x-auto">
                  {mode.example}
                </pre>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Quick Reference Footer */}
            <div className="px-5 pb-5">
                <div
                    className="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-900/30 rounded-lg">
                    <h4 className="font-medium text-amber-900 dark:text-amber-300 text-sm mb-2 flex items-center gap-2">
                        <Code2 size={16}/>
                        Quick Reference
                    </h4>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
                        <div className="flex items-center gap-2 text-amber-800 dark:text-amber-400">
                            <span
                                className="font-mono bg-amber-100 dark:bg-amber-900/40 px-1.5 py-0.5 rounded">{'{{env.*}}'}</span>
                            <span>Environment/workflow variables</span>
                        </div>
                        <div className="flex items-center gap-2 text-amber-800 dark:text-amber-400">
                            <span
                                className="font-mono bg-amber-100 dark:bg-amber-900/40 px-1.5 py-0.5 rounded">{'{{input.*}}'}</span>
                            <span>Parent node outputs</span>
                        </div>
                        <div className="flex items-center gap-2 text-amber-800 dark:text-amber-400">
                            <span
                                className="font-mono bg-amber-100 dark:bg-amber-900/40 px-1.5 py-0.5 rounded">{'{{nodeId.*}}'}</span>
                            <span>Specific node output</span>
                        </div>
                        <div className="flex items-center gap-2 text-amber-800 dark:text-amber-400">
                            <span
                                className="font-mono bg-amber-100 dark:bg-amber-900/40 px-1.5 py-0.5 rounded">{'{{input.arr[0]}}'}</span>
                            <span>Array index access</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );

    if (isModal) {
        return (
            <div
                className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
                onClick={(e) => {
                    if (e.target === e.currentTarget && onClose) {
                        onClose();
                    }
                }}
            >
                <div className="w-full max-w-3xl animate-in fade-in zoom-in-95 duration-200">
                    {content}
                </div>
            </div>
        );
    }

    return content;
};
