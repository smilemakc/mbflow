import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {
    AlertCircle,
    ArrowLeft,
    FileJson,
    Hash,
    Info,
    Layers,
    Loader2,
    Play,
    RefreshCw,
    Workflow,
    XCircle
} from 'lucide-react';
import {useTranslation} from '@/store/translations';
import {executionService} from '@/services/executionService';
import {Execution, NodeExecution} from '@/types/execution';
import {useToast} from '@/hooks/useToast';
import {useExecutionData} from '@/hooks/useExecutionData';
import {useExecutionWebSocket} from '@/hooks/useExecutionWebSocket';
import {
    calculateStats,
    CopyButton,
    ExecutionStatsGrid,
    formatDate,
    getStatusBadgeClass,
    getStatusIcon,
    JsonViewer,
    NodeExecutionCard,
    WebSocketStatusBadge
} from '@/components/execution';
import {Button} from '@/components/ui';

export const ExecutionDetailPage: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const navigate = useNavigate();
    const t = useTranslation();
    const {showToast} = useToast();

    const [isRetrying, setIsRetrying] = useState(false);
    const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
    const [activeTab, setActiveTab] = useState<'nodes' | 'overview'>('nodes');

    const {execution, workflow, isLoading, refetch, setExecution} = useExecutionData(id);

    const handleExecutionUpdate = useCallback((updates: Partial<Execution>) => {
        setExecution(prev => prev ? {...prev, ...updates} : prev);
    }, [setExecution]);

    const handleNodeUpdate = useCallback((nodeId: string, nodeExec: NodeExecution) => {
        setExecution(prev => {
            if (!prev) return prev;

            const existingIndex = prev.node_executions?.findIndex(ne => ne.node_id === nodeId) ?? -1;
            const updatedNodeExecs = [...(prev.node_executions || [])];

            if (existingIndex >= 0) {
                updatedNodeExecs[existingIndex] = {
                    ...updatedNodeExecs[existingIndex],
                    ...nodeExec,
                    input: nodeExec.input || updatedNodeExecs[existingIndex].input,
                    output: nodeExec.output || updatedNodeExecs[existingIndex].output
                };
            } else {
                updatedNodeExecs.push(nodeExec);
            }

            return {...prev, node_executions: updatedNodeExecs};
        });
    }, [setExecution]);

    const {wsConnected, connect, disconnect} = useExecutionWebSocket({
        executionId: id,
        onExecutionUpdate: handleExecutionUpdate,
        onNodeUpdate: handleNodeUpdate
    });

    const sortedNodeExecutions = useMemo(() => {
        if (!execution?.node_executions) return [];

        return [...execution.node_executions].sort((a, b) => {
            const timeA = new Date(a.started_at).getTime();
            const timeB = new Date(b.started_at).getTime();
            return timeA - timeB;
        });
    }, [execution?.node_executions]);

    useEffect(() => {
        if (execution) {
            if (execution.status === 'running' || execution.status === 'pending') {
                if (id) {
                    connect(id);
                }
            } else {
                disconnect();
            }
        }
    }, [execution?.status, id, connect, disconnect]);

    const handleRetry = async () => {
        if (!execution) return;

        setIsRetrying(true);
        try {
            await executionService.retry(execution.id);
            showToast({type: 'success', title: t.executionDetail?.retryStarted || 'Retry started'});
            await refetch();
        } catch (error) {
            console.error('Failed to retry execution:', error);
            showToast({type: 'error', title: t.executionDetail?.retryFailed || 'Failed to retry execution'});
        } finally {
            setIsRetrying(false);
        }
    };

    const toggleNodeExpanded = (nodeId: string) => {
        setExpandedNodes(prev => {
            const next = new Set(prev);
            if (next.has(nodeId)) {
                next.delete(nodeId);
            } else {
                next.add(nodeId);
            }
            return next;
        });
    };

    const expandAllNodes = () => {
        setExpandedNodes(new Set(sortedNodeExecutions.map(ne => ne.node_id)));
    };

    const collapseAllNodes = () => {
        setExpandedNodes(new Set());
    };

    const getStatusText = (status: string) => {
        return t.executions?.status?.[status as keyof typeof t.executions.status] || status;
    };

    const stats = calculateStats(execution?.node_executions);

    if (isLoading) {
        return (
            <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
                <div className="max-w-6xl mx-auto">
                    <div className="flex items-center justify-center py-24">
                        <Loader2 className="animate-spin text-blue-600 dark:text-blue-400" size={40}/>
                    </div>
                </div>
            </div>
        );
    }

    if (!execution) {
        return (
            <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
                <div className="max-w-6xl mx-auto">
                    <div className="flex flex-col items-center justify-center py-24 text-slate-500 dark:text-slate-400">
                        <AlertCircle size={48} className="mb-4"/>
                        <p className="text-lg font-medium">{t.executionDetail?.notFound || 'Execution not found'}</p>
                        <Button
                            onClick={() => navigate('/executions')}
                            variant="primary"
                            className="mt-4"
                        >
                            {t.executionDetail?.backToList || 'Back to Executions'}
                        </Button>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-6xl mx-auto space-y-6">

                {/* Back Button & Header */}
                <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
                    <div>
                        <Button
                            onClick={() => navigate('/executions')}
                            variant="ghost"
                            size="sm"
                            icon={<ArrowLeft size={16}/>}
                            className="mb-3"
                        >
                            {t.executionDetail?.backToList || 'Back to Executions'}
                        </Button>

                        <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-3">
                            {t.executionDetail?.title || 'Execution Details'}
                            <span
                                className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-sm font-medium border ${getStatusBadgeClass(execution.status)}`}>
                {getStatusIcon(execution.status)}
                                {getStatusText(execution.status)}
              </span>
                        </h1>

                        <div className="flex items-center gap-4 mt-2 text-sm text-slate-500 dark:text-slate-400">
              <span className="flex items-center gap-1.5 font-mono">
                <Hash size={14}/>
                  {execution.id.substring(0, 8)}...
              </span>
                            <span className="flex items-center gap-1.5">
                <Workflow size={14}/>
                                {workflow?.name || execution.workflow_id.substring(0, 8)}
              </span>
                        </div>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-3">
                        {(execution.status === 'running' || execution.status === 'pending') && (
                            <WebSocketStatusBadge connected={wsConnected}/>
                        )}

                        <Button
                            onClick={() => refetch(true)}
                            variant="outline"
                            size="sm"
                            icon={<RefreshCw size={16}/>}
                        >
                            {t.executionDetail?.refresh || 'Refresh'}
                        </Button>

                        {execution.status === 'failed' && (
                            <Button
                                onClick={handleRetry}
                                variant="primary"
                                size="sm"
                                loading={isRetrying}
                                icon={<Play size={16}/>}
                            >
                                {t.executions?.actions?.retry || 'Retry'}
                            </Button>
                        )}
                    </div>
                </div>

                <ExecutionStatsGrid execution={execution}/>

                {/* Execution Error Banner */}
                {execution.error && (
                    <div
                        className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-900/30 rounded-xl p-5">
                        <div className="flex items-start gap-4">
                            <XCircle className="text-red-600 dark:text-red-400 mt-0.5 shrink-0" size={24}/>
                            <div className="flex-1 min-w-0">
                                <h3 className="text-base font-semibold text-red-900 dark:text-red-300 mb-2">
                                    {t.executionDetail?.executionError || 'Execution Error'}
                                </h3>
                                <pre
                                    className="text-sm text-red-800 dark:text-red-400 font-mono whitespace-pre-wrap break-words bg-red-100 dark:bg-red-900/30 rounded-lg p-3">
                  {execution.error}
                </pre>
                            </div>
                            <CopyButton text={execution.error}/>
                        </div>
                    </div>
                )}

                {/* Tabs */}
                <div className="border-b border-slate-200 dark:border-slate-800">
                    <nav className="flex space-x-8">
                        <button
                            onClick={() => setActiveTab('nodes')}
                            className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${
                                activeTab === 'nodes'
                                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                    : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                            }`}
                        >
                            <div className="flex items-center gap-2">
                                <Layers size={16}/>
                                {t.executionDetail?.nodeExecutions || 'Node Executions'}
                                <span
                                    className="bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 px-2 py-0.5 rounded-full text-xs">
                  {stats.total}
                </span>
                            </div>
                        </button>

                        <button
                            onClick={() => setActiveTab('overview')}
                            className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${
                                activeTab === 'overview'
                                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                    : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                            }`}
                        >
                            <div className="flex items-center gap-2">
                                <Info size={16}/>
                                {t.executionDetail?.overview || 'Overview'}
                            </div>
                        </button>
                    </nav>
                </div>

                {/* Tab Content */}
                {activeTab === 'nodes' && (
                    <div className="space-y-4">
                        {/* Expand/Collapse Controls */}
                        <div className="flex items-center justify-between">
                            <p className="text-sm text-slate-500 dark:text-slate-400">
                                {t.executionDetail?.nodeExecutionsDesc || 'Click on a node to view its inputs and outputs'}
                            </p>
                            <div className="flex items-center gap-2">
                                <Button
                                    onClick={expandAllNodes}
                                    variant="ghost"
                                    size="sm"
                                >
                                    {t.executionDetail?.expandAll || 'Expand All'}
                                </Button>
                                <Button
                                    onClick={collapseAllNodes}
                                    variant="ghost"
                                    size="sm"
                                >
                                    {t.executionDetail?.collapseAll || 'Collapse All'}
                                </Button>
                            </div>
                        </div>

                        {/* Node Execution Cards */}
                        {sortedNodeExecutions.length > 0 ? (
                            <div className="space-y-3">
                                {sortedNodeExecutions.map((nodeExec, index) => (
                                    <NodeExecutionCard
                                        key={nodeExec.id || nodeExec.node_id}
                                        nodeExec={nodeExec}
                                        index={index}
                                        isExpanded={expandedNodes.has(nodeExec.node_id)}
                                        onToggle={() => toggleNodeExpanded(nodeExec.node_id)}
                                    />
                                ))}
                            </div>
                        ) : (
                            <div
                                className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-8 text-center">
                                <Layers size={40} className="mx-auto text-slate-300 dark:text-slate-600 mb-3"/>
                                <p className="text-slate-500 dark:text-slate-400">
                                    {t.executionDetail?.noNodeExecutions || 'No node executions found'}
                                </p>
                            </div>
                        )}
                    </div>
                )}

                {activeTab === 'overview' && (
                    <div className="space-y-6">
                        {/* Metadata Grid */}
                        <div
                            className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                                {t.executionDetail?.metadata || 'Metadata'}
                            </h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div>
                                    <label
                                        className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        Execution ID
                                    </label>
                                    <div className="flex items-center gap-2 mt-1">
                                        <p className="text-sm font-mono text-slate-700 dark:text-slate-300 break-all">
                                            {execution.id}
                                        </p>
                                        <CopyButton text={execution.id}/>
                                    </div>
                                </div>
                                <div>
                                    <label
                                        className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        Workflow ID
                                    </label>
                                    <div className="flex items-center gap-2 mt-1">
                                        <p className="text-sm font-mono text-slate-700 dark:text-slate-300 break-all">
                                            {execution.workflow_id}
                                        </p>
                                        <CopyButton text={execution.workflow_id}/>
                                    </div>
                                </div>
                                <div>
                                    <label
                                        className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        {t.executions?.table?.triggeredBy || 'Triggered By'}
                                    </label>
                                    <p className="text-sm text-slate-700 dark:text-slate-300 mt-1">
                                        {execution.triggered_by || 'System'}
                                    </p>
                                </div>
                                <div>
                                    <label
                                        className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        {t.executionDetail?.completedAt || 'Completed At'}
                                    </label>
                                    <p className="text-sm text-slate-700 dark:text-slate-300 mt-1">
                                        {execution.completed_at ? formatDate(execution.completed_at) : '-'}
                                    </p>
                                </div>
                            </div>
                        </div>

                        {/* Input/Output */}
                        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                            <div
                                className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                                <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                                    <FileJson size={18}/>
                                    {t.executions?.details?.input || 'Input'}
                                </h3>
                                <JsonViewer
                                    data={execution.input}
                                    title=""
                                    defaultExpanded={true}
                                    maxHeight="400px"
                                />
                            </div>

                            <div
                                className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                                <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                                    <FileJson size={18}/>
                                    {t.executions?.details?.output || 'Output'}
                                </h3>
                                <JsonViewer
                                    data={execution.output}
                                    title=""
                                    defaultExpanded={true}
                                    maxHeight="400px"
                                />
                            </div>
                        </div>

                        {/* Variables */}
                        {execution.variables && Object.keys(execution.variables).length > 0 && (
                            <div
                                className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                                <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                                    {t.executionDetail?.variables || 'Variables'}
                                </h3>
                                <JsonViewer
                                    data={execution.variables}
                                    title=""
                                    defaultExpanded={true}
                                    maxHeight="300px"
                                />
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};
