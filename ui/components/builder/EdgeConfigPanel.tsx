import React, {useEffect, useState} from 'react';
import {useDagStore} from '@/store/dagStore.ts';
import {useTranslation} from '@/store/translations';
import {VariableAutocomplete} from "@/components/builder";
import {ArrowRight, Check, Trash2, X, XCircle} from 'lucide-react';
import {Button, ConfirmModal} from '@/components/ui';

export const EdgeConfigPanel: React.FC = () => {
    const {
        selectedEdgeId,
        edges,
        nodes,
        setSelectedEdgeId,
        updateEdge,
        deleteEdge,
    } = useDagStore();

    const t = useTranslation();

    const selectedEdge = edges.find((e) => e.id === selectedEdgeId);
    const isOpen = !!selectedEdge;

    const [sourceHandle, setSourceHandle] = useState<string>('');
    const [condition, setCondition] = useState<string>('');
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

    useEffect(() => {
        if (selectedEdge) {
            setSourceHandle(selectedEdge.sourceHandle || '');
            setCondition((selectedEdge.data as any)?.condition || '');
        } else {
            setSourceHandle('');
            setCondition('');
        }
    }, [selectedEdge]);

    if (!selectedEdge) return null;

    const sourceNode = nodes.find((n) => n.id === selectedEdge.source);
    const targetNode = nodes.find((n) => n.id === selectedEdge.target);

    const sourceNodeLabel = sourceNode?.data?.label || selectedEdge.source;
    const targetNodeLabel = targetNode?.data?.label || selectedEdge.target;

    const isFromConditionalNode = sourceNode?.data?.type === 'conditional';

    const handleSave = () => {
        if (!selectedEdge) return;

        updateEdge(selectedEdge.id, {
            sourceHandle: sourceHandle || undefined,
            data: {
                ...selectedEdge.data,
                condition: condition || undefined,
            },
        });

        setSelectedEdgeId(null);
    };

    const handleDelete = () => {
        if (!selectedEdge) return;
        deleteEdge(selectedEdge.id);
        setShowDeleteConfirm(false);
    };

    const handleClose = () => {
        setSelectedEdgeId(null);
    };

    return (
        <>
            {/* Panel */}
            <div
                className={`fixed right-0 top-0 h-full w-96 bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800 shadow-2xl z-50 transition-transform duration-300 ease-in-out transform ${
                    isOpen ? 'translate-x-0' : 'translate-x-full'
                }`}
            >
                <div className="flex flex-col h-full">
                    {/* Header */}
                    <div
                        className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 p-4 bg-slate-50/50 dark:bg-slate-900/50">
                        <div className="flex items-center gap-2">
                            <ArrowRight className="w-5 h-5 text-slate-700 dark:text-slate-300"/>
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                                {t.edge.title}
                            </h3>
                        </div>
                        <Button
                            variant="ghost"
                            size="sm"
                            icon={<X className="w-5 h-5"/>}
                            onClick={handleClose}
                        />
                    </div>

                    {/* Content */}
                    <div className="flex-1 overflow-y-auto p-4 space-y-4">
                        {/* Edge Info */}
                        <div className="rounded-lg bg-slate-50 dark:bg-slate-950 p-3">
                            <div className="mb-2 text-sm font-medium text-slate-700 dark:text-slate-300">
                                {t.edge.connection}
                            </div>
                            <div className="flex items-center gap-2 text-sm">
                <span
                    className="rounded bg-blue-100 dark:bg-blue-900/30 px-2 py-1 font-mono text-blue-700 dark:text-blue-300">
                  {sourceNodeLabel}
                </span>
                                <ArrowRight className="w-4 h-4 text-slate-400 dark:text-slate-500"/>
                                <span
                                    className="rounded bg-green-100 dark:bg-green-900/30 px-2 py-1 font-mono text-green-700 dark:text-green-300">
                  {targetNodeLabel}
                </span>
                            </div>
                        </div>

                        {/* Source Handle (for conditional nodes) */}
                        {isFromConditionalNode && (
                            <div className="space-y-2">
                                <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
                                    {t.edge.branch}
                                </label>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => setSourceHandle('true')}
                                        className={`flex-1 rounded-lg border-2 px-4 py-2 text-sm font-medium transition-colors ${
                                            sourceHandle === 'true'
                                                ? 'border-green-500 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400'
                                                : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-950 text-slate-600 dark:text-slate-400 hover:border-green-300 dark:hover:border-green-700'
                                        }`}
                                    >
                                        <Check className="inline w-4 h-4 mr-1"/>
                                        {t.edge.trueBranch}
                                    </button>
                                    <button
                                        onClick={() => setSourceHandle('false')}
                                        className={`flex-1 rounded-lg border-2 px-4 py-2 text-sm font-medium transition-colors ${
                                            sourceHandle === 'false'
                                                ? 'border-red-500 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400'
                                                : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-950 text-slate-600 dark:text-slate-400 hover:border-red-300 dark:hover:border-red-700'
                                        }`}
                                    >
                                        <XCircle className="inline w-4 h-4 mr-1"/>
                                        {t.edge.falseBranch}
                                    </button>
                                </div>
                                <p className="text-xs text-slate-500 dark:text-slate-400">
                                    {t.edge.branchDescription}
                                </p>
                            </div>
                        )}

                        {/* Condition Expression */}
                        <div className="space-y-2">
                            <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
                                {t.edge.conditionExpression}{' '}
                                <span className="text-slate-400 dark:text-slate-500">{t.edge.optional}</span>
                            </label>
                            <VariableAutocomplete
                                type="textarea"
                                value={condition}
                                onChange={setCondition}
                                rows={3}
                                placeholder={t.edge.conditionPlaceholder}
                                className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-none text-slate-800 dark:text-slate-200 placeholder-slate-400 font-mono"
                            />
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                {t.edge.conditionDescription}
                            </p>
                        </div>

                        {/* Info Box */}
                        <div
                            className="rounded-lg border border-blue-200 dark:border-blue-900/50 bg-blue-50 dark:bg-blue-900/20 p-3">
                            <h4 className="mb-2 text-sm font-semibold text-blue-800 dark:text-blue-300">
                                {t.edge.conditionExamples}
                            </h4>
                            <ul className="space-y-1 text-xs text-blue-700 dark:text-blue-400">
                                <li>
                                    <code className="rounded bg-white dark:bg-blue-950/50 px-1 py-0.5">
                                        output.success == true
                                    </code>
                                </li>
                                <li>
                                    <code className="rounded bg-white dark:bg-blue-950/50 px-1 py-0.5">
                                        output.count &gt; 0
                                    </code>
                                </li>
                                <li>
                                    <code className="rounded bg-white dark:bg-blue-950/50 px-1 py-0.5">
                                        output.status == &quot;completed&quot;
                                    </code>
                                </li>
                            </ul>
                        </div>
                    </div>

                    {/* Footer */}
                    <div
                        className="space-y-2 border-t border-slate-200 dark:border-slate-800 p-4 bg-slate-50/50 dark:bg-slate-900/50">
                        <div className="flex gap-2">
                            <Button
                                variant="primary"
                                onClick={handleSave}
                                className="flex-1"
                            >
                                {t.edge.saveChanges}
                            </Button>
                            <Button
                                variant="outline"
                                onClick={handleClose}
                            >
                                {t.common.cancel}
                            </Button>
                        </div>
                        <Button
                            variant="danger"
                            fullWidth
                            icon={<Trash2 size={16} />}
                            onClick={() => setShowDeleteConfirm(true)}
                        >
                            {t.edge.deleteEdge}
                        </Button>
                    </div>
                </div>
            </div>

            {/* Delete Confirmation Modal */}
            <ConfirmModal
                isOpen={showDeleteConfirm}
                onClose={() => setShowDeleteConfirm(false)}
                onConfirm={handleDelete}
                title={t.edge.deleteModal.title}
                message={t.edge.deleteModal.message}
                confirmText={t.edge.deleteModal.confirm}
                variant="danger"
            />

            {/* Backdrop */}
            {isOpen && (
                <div
                    className="fixed inset-0 z-40 bg-black/20 dark:bg-black/40 transition-opacity"
                    onClick={handleClose}
                />
            )}
        </>
    );
};
