import React, {memo} from 'react';
import {Handle, NodeProps, Position} from 'reactflow';
import {
    AlertCircle,
    Bot,
    CheckCircle2,
    Clock,
    Clock3,
    GitBranch,
    Globe,
    HelpCircle,
    Loader2,
    MoreHorizontal,
    Sparkles
} from 'lucide-react';
import {NodeData, NodeStatus, NodeType} from '@/types';

// Helper to get color/icon based on type
const getNodeStyles = (type?: NodeType) => {
    switch (type) {
        case NodeType.TELEGRAM:
            return {
                icon: Bot,
                color: 'text-blue-500',
                bg: 'bg-blue-500',
                border: 'border-blue-200 dark:border-blue-900',
                gradient: 'from-blue-50 to-white dark:from-blue-900/20 dark:to-slate-900'
            };
        case NodeType.LLM:
            return {
                icon: Sparkles,
                color: 'text-purple-500',
                bg: 'bg-purple-500',
                border: 'border-purple-200 dark:border-purple-900',
                gradient: 'from-purple-50 to-white dark:from-purple-900/20 dark:to-slate-900'
            };
        case NodeType.DELAY:
            return {
                icon: Clock,
                color: 'text-orange-500',
                bg: 'bg-orange-500',
                border: 'border-orange-200 dark:border-orange-900',
                gradient: 'from-orange-50 to-white dark:from-orange-900/20 dark:to-slate-900'
            };
        case NodeType.HTTP:
            return {
                icon: Globe,
                color: 'text-emerald-500',
                bg: 'bg-emerald-500',
                border: 'border-emerald-200 dark:border-emerald-900',
                gradient: 'from-emerald-50 to-white dark:from-emerald-900/20 dark:to-slate-900'
            };
        case NodeType.CONDITIONAL:
            return {
                icon: GitBranch,
                color: 'text-slate-500',
                bg: 'bg-slate-500',
                border: 'border-slate-300 dark:border-slate-700',
                gradient: 'from-slate-100 to-white dark:from-slate-800 dark:to-slate-900'
            };
        default:
            return {
                icon: HelpCircle,
                color: 'text-slate-400',
                bg: 'bg-slate-400',
                border: 'border-slate-200 dark:border-slate-800',
                gradient: 'from-white to-slate-50 dark:from-slate-900 dark:to-slate-950'
            };
    }
};

const StatusIcon = ({status}: { status?: NodeStatus }) => {
    if (!status || status === NodeStatus.IDLE || status === NodeStatus.PENDING) return null;

    switch (status) {
        case NodeStatus.RUNNING:
            return <Loader2 size={14} className="text-blue-500 animate-spin"/>;
        case NodeStatus.SUCCESS:
            return <CheckCircle2 size={14} className="text-green-500"/>;
        case NodeStatus.ERROR:
            return <AlertCircle size={14} className="text-red-500"/>;
        case NodeStatus.SKIPPED:
            return <div className="w-3 h-3 rounded-full bg-slate-300 dark:bg-slate-600"/>;
        default:
            return null;
    }
};

const StatusBorder = ({status}: { status?: NodeStatus }) => {
    if (status === NodeStatus.RUNNING) return 'ring-2 ring-blue-400 dark:ring-blue-500 ring-offset-2 dark:ring-offset-slate-950 animate-pulse';
    if (status === NodeStatus.SUCCESS) return 'ring-1 ring-green-500/50 dark:ring-green-500/50';
    if (status === NodeStatus.ERROR) return 'ring-2 ring-red-500/50 dark:ring-red-500/50';
    return '';
};

export const CustomNode = memo(({data, selected}: NodeProps<NodeData>) => {
    const styles = getNodeStyles(data.type);
    const Icon = styles.icon;
    const statusClass = StatusBorder({status: data.status});

    return (
        <div
            className={`
        relative min-w-[240px] rounded-xl border bg-white dark:bg-slate-900 transition-all duration-200 group
        ${selected ? 'ring-2 ring-blue-500 border-transparent shadow-lg shadow-blue-500/20' : `border-slate-200 dark:border-slate-800 hover:border-blue-300 dark:hover:border-blue-700 hover:shadow-md`}
        ${statusClass}
      `}
        >
            {/* Handles */}
            <Handle
                type="target"
                position={Position.Top}
                className="!w-3 !h-3 !bg-slate-400 dark:!bg-slate-600 !border-2 !border-white dark:!border-slate-900 group-hover:!bg-blue-500 transition-colors"
            />

            {/* Header */}
            <div className={`
        flex items-center justify-between p-3 rounded-t-xl border-b border-slate-100 dark:border-slate-800 bg-gradient-to-b ${styles.gradient}
      `}>
                <div className="flex items-center gap-2">
                    <div
                        className={`p-1.5 rounded-lg bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-800`}>
                        <Icon size={16} className={styles.color}/>
                    </div>
                    <div className="flex flex-col">
            <span className="text-xs font-bold text-slate-700 dark:text-slate-200 leading-tight">
              {data.label}
            </span>
                        <span
                            className="text-[10px] font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wider">
              {data.type || 'Node'}
            </span>
                    </div>
                </div>
                <div className="flex items-center gap-1">
                    <StatusIcon status={data.status}/>
                    <button
                        className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity">
                        <MoreHorizontal size={14}/>
                    </button>
                </div>
            </div>

            {/* Body */}
            <div className="p-3">
                <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2 leading-relaxed">
                    {data.description || 'No description provided.'}
                </p>

                {/* Dynamic content snippet based on type */}
                <div className="flex items-center justify-between mt-2">
                    {data.type === NodeType.DELAY && data.config?.cron && (
                        <div
                            className="text-[10px] font-mono bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 px-2 py-1 rounded w-fit">
                            {data.config.cron}
                        </div>
                    )}

                    {/* Last Run Duration */}
                    {data.lastRun && data.status !== NodeStatus.RUNNING && (
                        <div className="flex items-center text-[10px] text-slate-400 ml-auto" title="Last run duration">
                            <Clock3 size={10} className="mr-1"/>
                            {data.lastRun.duration}ms
                        </div>
                    )}
                </div>
            </div>

            <Handle
                type="source"
                position={Position.Bottom}
                className="!w-3 !h-3 !bg-slate-400 dark:!bg-slate-600 !border-2 !border-white dark:!border-slate-900 group-hover:!bg-blue-500 transition-colors"
            />
        </div>
    );
});
