import React, {useState} from 'react';
import {useDagStore} from '@/store/dagStore';
import {useTranslation} from '@/store/translations';
import {BookOpen, Check, Copy, Eye, EyeOff, Info, Plus, Trash2, Variable, X} from 'lucide-react';
import {WorkflowVariablesGuide} from '@/components/builder';
import {Button} from '../ui';

interface WorkflowVariablesEditorProps {
    isOpen: boolean;
    onClose: () => void;
}

export const WorkflowVariablesEditor: React.FC<WorkflowVariablesEditorProps> = ({
                                                                                    isOpen,
                                                                                    onClose
                                                                                }) => {
    const {workflowVariables, updateWorkflowVariables} = useDagStore();
    const t = useTranslation();

    const [newKey, setNewKey] = useState('');
    const [newValue, setNewValue] = useState('');
    const [editingKey, setEditingKey] = useState<string | null>(null);
    const [editValue, setEditValue] = useState('');
    const [copiedKey, setCopiedKey] = useState<string | null>(null);
    const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});
    const [showVariablesGuide, setShowVariablesGuide] = useState(false);

    const variables = Object.entries(workflowVariables);

    const handleAdd = () => {
        if (!newKey.trim()) return;

        const key = newKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
        updateWorkflowVariables({
            ...workflowVariables,
            [key]: newValue
        });
        setNewKey('');
        setNewValue('');
    };

    const handleUpdate = (key: string) => {
        updateWorkflowVariables({
            ...workflowVariables,
            [key]: editValue
        });
        setEditingKey(null);
        setEditValue('');
    };

    const handleDelete = (key: string) => {
        const newVars = {...workflowVariables};
        delete newVars[key];
        updateWorkflowVariables(newVars);
    };

    const copyTemplate = (key: string) => {
        navigator.clipboard.writeText(`{{env.${key}}}`);
        setCopiedKey(key);
        setTimeout(() => setCopiedKey(null), 2000);
    };

    const isSecret = (key: string) => {
        const secretPatterns = ['TOKEN', 'SECRET', 'KEY', 'PASSWORD', 'API_KEY', 'APIKEY', 'PASS', 'AUTH'];
        return secretPatterns.some(pattern => key.toUpperCase().includes(pattern));
    };

    const toggleShowSecret = (key: string) => {
        setShowSecrets(prev => ({...prev, [key]: !prev[key]}));
    };

    const maskValue = (value: string) => {
        if (value.length <= 4) return '••••••••';
        return value.slice(0, 2) + '••••••' + value.slice(-2);
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div
                className="w-full max-w-2xl max-h-[80vh] bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden flex flex-col">
                {/* Header */}
                <div
                    className="p-4 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center bg-slate-50 dark:bg-slate-800/50">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg">
                            <Variable size={20} className="text-indigo-600 dark:text-indigo-400"/>
                        </div>
                        <div>
                            <h2 className="font-bold text-slate-800 dark:text-slate-100">
                                Workflow Variables
                            </h2>
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                Define environment variables for this workflow
                            </p>
                        </div>
                    </div>
                    <Button
                        variant={showVariablesGuide ? 'primary' : 'outline'}
                        size="sm"
                        icon={<BookOpen size={16} />}
                        onClick={() => setShowVariablesGuide(!showVariablesGuide)}
                        className="shadow-sm"
                    >
                        Variables Guide
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={<X size={18} />}
                        onClick={onClose}
                    />
                </div>

                {showVariablesGuide && (
                    <WorkflowVariablesGuide
                        isModal={true}
                        onClose={() => setShowVariablesGuide(false)}
                    />
                )}

                {/* Info Banner */}
                <div
                    className="mx-4 mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/30 rounded-lg">
                    <div className="flex gap-2 text-sm">
                        <Info size={16} className="text-blue-600 dark:text-blue-400 shrink-0 mt-0.5"/>
                        <p className="text-blue-700 dark:text-blue-300">
                            Use <code
                            className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900/40 rounded text-xs font-mono">{'{{env.VARIABLE_NAME}}'}</code> in
                            node configurations to reference these variables.
                        </p>
                    </div>
                </div>

                {/* Variables List */}
                <div className="flex-1 overflow-y-auto p-4 space-y-2">
                    {variables.length === 0 ? (
                        <div className="text-center py-8 text-slate-400">
                            <Variable size={32} className="mx-auto mb-2 opacity-50"/>
                            <p className="text-sm">No variables defined yet</p>
                            <p className="text-xs mt-1">Add your first variable below</p>
                        </div>
                    ) : (
                        variables.map(([key, value]) => (
                            <div
                                key={key}
                                className="flex items-center gap-2 p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg group"
                            >
                                {editingKey === key ? (
                                    // Edit mode
                                    <>
                                        <div className="flex-1 flex gap-2">
                      <span
                          className="px-2 py-1.5 bg-slate-200 dark:bg-slate-700 rounded text-sm font-mono text-slate-700 dark:text-slate-300 min-w-[120px]">
                        {key}
                      </span>
                                            <input
                                                type="text"
                                                value={editValue}
                                                onChange={(e) => setEditValue(e.target.value)}
                                                onKeyDown={(e) => e.key === 'Enter' && handleUpdate(key)}
                                                className="flex-1 px-3 py-1.5 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-600 rounded text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none"
                                                autoFocus
                                            />
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            icon={<Check size={16}/>}
                                            onClick={() => handleUpdate(key)}
                                            className="bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 hover:bg-green-200 dark:hover:bg-green-900/50"
                                        />
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            icon={<X size={16}/>}
                                            onClick={() => setEditingKey(null)}
                                        />
                                    </>
                                ) : (
                                    // View mode
                                    <>
                    <span
                        className="px-2 py-1.5 bg-slate-200 dark:bg-slate-700 rounded text-sm font-mono text-slate-700 dark:text-slate-300 min-w-[120px]">
                      {key}
                    </span>
                                        <span
                                            className="flex-1 text-sm text-slate-600 dark:text-slate-400 font-mono truncate">
                      {isSecret(key) && !showSecrets[key] ? maskValue(value) : value || '(empty)'}
                    </span>

                                        {/* Actions */}
                                        <div
                                            className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                            {isSecret(key) && (
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    icon={showSecrets[key] ? <EyeOff size={14}/> : <Eye size={14}/>}
                                                    onClick={() => toggleShowSecret(key)}
                                                    title={showSecrets[key] ? 'Hide value' : 'Show value'}
                                                />
                                            )}
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                icon={copiedKey === key ? <Check size={14} className="text-green-500"/> : <Copy size={14}/>}
                                                onClick={() => copyTemplate(key)}
                                                title="Copy template"
                                            />
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => {
                                                    setEditingKey(key);
                                                    setEditValue(value);
                                                }}
                                                title="Edit"
                                                className="text-blue-500 hover:text-blue-600 hover:bg-blue-100 dark:hover:bg-blue-900/30"
                                            >
                                                Edit
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                icon={<Trash2 size={14}/>}
                                                onClick={() => handleDelete(key)}
                                                title="Delete"
                                                className="text-red-500 hover:text-red-600 hover:bg-red-100 dark:hover:bg-red-900/30"
                                            />
                                        </div>
                                    </>
                                )}
                            </div>
                        ))
                    )}
                </div>

                {/* Add New Variable */}
                <div className="p-4 border-t border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50">
                    <div className="flex gap-2">
                        <input
                            type="text"
                            value={newKey}
                            onChange={(e) => setNewKey(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, '_'))}
                            placeholder="VARIABLE_NAME"
                            className="w-40 px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm font-mono focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none"
                        />
                        <input
                            type="text"
                            value={newValue}
                            onChange={(e) => setNewValue(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
                            placeholder="Value..."
                            className="flex-1 px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none"
                        />
                        <Button
                            variant="primary"
                            icon={<Plus size={16}/>}
                            onClick={handleAdd}
                            disabled={!newKey.trim()}
                        >
                            Add
                        </Button>
                    </div>
                    <p className="mt-2 text-xs text-slate-400">
                        Variable names should be UPPERCASE with underscores (e.g., API_KEY, BASE_URL)
                    </p>
                </div>
            </div>
        </div>
    );
};
