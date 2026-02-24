import React from 'react';
import {Calendar, CheckCircle, Layers, Timer, XCircle} from 'lucide-react';
import {useTranslation} from '@/store/translations';
import {Execution} from '@/types/execution';
import {calculateStats, formatDate, formatDuration} from '@/components/execution';

interface ExecutionStatsGridProps {
    execution: Execution;
}

export const ExecutionStatsGrid: React.FC<ExecutionStatsGridProps> = ({execution}) => {
    const t = useTranslation();
    const stats = calculateStats(execution.node_executions);

    return (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
                <div className="flex items-center gap-2 text-slate-500 dark:text-slate-400 mb-1">
                    <Layers size={16}/>
                    <span className="text-xs font-medium uppercase tracking-wider">
                        {t.executionDetail?.totalNodes || 'Total Nodes'}
                    </span>
                </div>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">{stats.total}</p>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
                <div className="flex items-center gap-2 text-green-500 mb-1">
                    <CheckCircle size={16}/>
                    <span className="text-xs font-medium uppercase tracking-wider">
                        {t.executions?.status?.completed || 'Completed'}
                    </span>
                </div>
                <p className="text-2xl font-bold text-green-600 dark:text-green-400">{stats.completed}</p>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
                <div className="flex items-center gap-2 text-red-500 mb-1">
                    <XCircle size={16}/>
                    <span className="text-xs font-medium uppercase tracking-wider">
                        {t.executions?.status?.failed || 'Failed'}
                    </span>
                </div>
                <p className="text-2xl font-bold text-red-600 dark:text-red-400">{stats.failed}</p>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
                <div className="flex items-center gap-2 text-slate-500 dark:text-slate-400 mb-1">
                    <Calendar size={16}/>
                    <span className="text-xs font-medium uppercase tracking-wider">
                        {t.executionDetail?.startedAt || 'Started'}
                    </span>
                </div>
                <p className="text-sm font-medium text-slate-900 dark:text-white">
                    {formatDate(execution.started_at)}
                </p>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
                <div className="flex items-center gap-2 text-slate-500 dark:text-slate-400 mb-1">
                    <Timer size={16}/>
                    <span className="text-xs font-medium uppercase tracking-wider">
                        {t.executions?.table?.duration || 'Duration'}
                    </span>
                </div>
                <p className="text-2xl font-bold font-mono text-slate-900 dark:text-white">
                    {formatDuration(execution.started_at, execution.completed_at)}
                </p>
            </div>
        </div>
    );
};
