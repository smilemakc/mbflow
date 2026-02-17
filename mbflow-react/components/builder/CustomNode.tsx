import React, {memo} from 'react';
import {Handle, NodeProps, Position} from 'reactflow';
import {
    AlertCircle,
    Box,
    Braces,
    BrushCleaning,
    CheckCircle,
    CheckCircle2,
    Clock,
    Clock3,
    Code,
    Download,
    FileDown,
    FileSearch,
    FileText,
    FileUp,
    Folder,
    GitBranch,
    GitMerge,
    Globe,
    HardDrive,
    HelpCircle,
    Loader2,
    Lock,
    MessageSquare,
    MoreHorizontal,
    Network,
    Rss,
    Send,
    Sheet,
    Sparkles,
    Table,
    Unlock,
    Zap,
} from 'lucide-react';
import {NodeData, NodeStatus, NodeType} from '@/types';
import {SubWorkflowProgress} from './SubWorkflowProgress';

// Helper to get color/icon based on type
const getNodeStyles = (type?: NodeType) => {
    switch (type) {
        // Telegram nodes
        case NodeType.TELEGRAM:
            return {
                icon: Send,
                color: 'text-sky-500',
                bg: 'bg-sky-500',
                border: 'border-sky-200 dark:border-sky-900',
                gradient: 'from-sky-50 to-white dark:from-sky-900/20 dark:to-slate-900'
            };
        case NodeType.TELEGRAM_DOWNLOAD:
            return {
                icon: Download,
                color: 'text-sky-500',
                bg: 'bg-sky-500',
                border: 'border-sky-200 dark:border-sky-900',
                gradient: 'from-sky-50 to-white dark:from-sky-900/20 dark:to-slate-900'
            };
        case NodeType.TELEGRAM_PARSE:
            return {
                icon: FileSearch,
                color: 'text-sky-500',
                bg: 'bg-sky-500',
                border: 'border-sky-200 dark:border-sky-900',
                gradient: 'from-sky-50 to-white dark:from-sky-900/20 dark:to-slate-900'
            };
        case NodeType.TELEGRAM_CALLBACK:
            return {
                icon: CheckCircle,
                color: 'text-sky-500',
                bg: 'bg-sky-500',
                border: 'border-sky-200 dark:border-sky-900',
                gradient: 'from-sky-50 to-white dark:from-sky-900/20 dark:to-slate-900'
            };

        // Actions
        case NodeType.LLM:
            return {
                icon: Sparkles,
                color: 'text-purple-500',
                bg: 'bg-purple-500',
                border: 'border-purple-200 dark:border-purple-900',
                gradient: 'from-purple-50 to-white dark:from-purple-900/20 dark:to-slate-900'
            };
        case NodeType.HTTP:
            return {
                icon: Globe,
                color: 'text-green-500',
                bg: 'bg-green-500',
                border: 'border-green-200 dark:border-green-900',
                gradient: 'from-green-50 to-white dark:from-green-900/20 dark:to-slate-900'
            };
        case NodeType.TRANSFORM:
            return {
                icon: Zap,
                color: 'text-amber-500',
                bg: 'bg-amber-500',
                border: 'border-amber-200 dark:border-amber-900',
                gradient: 'from-amber-50 to-white dark:from-amber-900/20 dark:to-slate-900'
            };
        case NodeType.FUNCTION_CALL:
            return {
                icon: Code,
                color: 'text-blue-500',
                bg: 'bg-blue-500',
                border: 'border-blue-200 dark:border-blue-900',
                gradient: 'from-blue-50 to-white dark:from-blue-900/20 dark:to-slate-900'
            };
        case NodeType.RSS_PARSER:
            return {
                icon: Rss,
                color: 'text-orange-500',
                bg: 'bg-orange-500',
                border: 'border-orange-200 dark:border-orange-900',
                gradient: 'from-orange-50 to-white dark:from-orange-900/20 dark:to-slate-900'
            };
        case NodeType.GOOGLE_SHEETS:
            return {
                icon: Sheet,
                color: 'text-green-600',
                bg: 'bg-green-600',
                border: 'border-green-200 dark:border-green-900',
                gradient: 'from-green-50 to-white dark:from-green-900/20 dark:to-slate-900'
            };
        case NodeType.GOOGLE_DRIVE:
            return {
                icon: HardDrive,
                color: 'text-blue-500',
                bg: 'bg-blue-500',
                border: 'border-blue-200 dark:border-blue-900',
                gradient: 'from-blue-50 to-white dark:from-blue-900/20 dark:to-slate-900'
            };

        // Triggers
        case NodeType.DELAY:
            return {
                icon: Clock,
                color: 'text-cyan-500',
                bg: 'bg-cyan-500',
                border: 'border-cyan-200 dark:border-cyan-900',
                gradient: 'from-cyan-50 to-white dark:from-cyan-900/20 dark:to-slate-900'
            };

        // Logic
        case NodeType.CONDITIONAL:
            return {
                icon: GitBranch,
                color: 'text-pink-500',
                bg: 'bg-pink-500',
                border: 'border-pink-200 dark:border-pink-900',
                gradient: 'from-pink-50 to-white dark:from-pink-900/20 dark:to-slate-900'
            };
        case NodeType.MERGE:
            return {
                icon: GitMerge,
                color: 'text-violet-500',
                bg: 'bg-violet-500',
                border: 'border-violet-200 dark:border-violet-900',
                gradient: 'from-violet-50 to-white dark:from-violet-900/20 dark:to-slate-900'
            };

        // Storage
        case NodeType.FILE_STORAGE:
            return {
                icon: Folder,
                color: 'text-teal-500',
                bg: 'bg-teal-500',
                border: 'border-teal-200 dark:border-teal-900',
                gradient: 'from-teal-50 to-white dark:from-teal-900/20 dark:to-slate-900'
            };

        // Adapters
        case NodeType.HTML_CLEAN:
            return {
                icon: BrushCleaning,
                color: 'text-orange-500',
                bg: 'bg-orange-500',
                border: 'border-orange-200 dark:border-orange-900',
                gradient: 'from-orange-50 to-white dark:from-orange-900/20 dark:to-slate-900'
            };
        case NodeType.BASE64_TO_BYTES:
            return {
                icon: Unlock,
                color: 'text-red-500',
                bg: 'bg-red-500',
                border: 'border-red-200 dark:border-red-900',
                gradient: 'from-red-50 to-white dark:from-red-900/20 dark:to-slate-900'
            };
        case NodeType.BYTES_TO_BASE64:
            return {
                icon: Lock,
                color: 'text-amber-500',
                bg: 'bg-amber-500',
                border: 'border-amber-200 dark:border-amber-900',
                gradient: 'from-amber-50 to-white dark:from-amber-900/20 dark:to-slate-900'
            };
        case NodeType.STRING_TO_JSON:
            return {
                icon: Braces,
                color: 'text-violet-500',
                bg: 'bg-violet-500',
                border: 'border-violet-200 dark:border-violet-900',
                gradient: 'from-violet-50 to-white dark:from-violet-900/20 dark:to-slate-900'
            };
        case NodeType.JSON_TO_STRING:
            return {
                icon: FileText,
                color: 'text-pink-500',
                bg: 'bg-pink-500',
                border: 'border-pink-200 dark:border-pink-900',
                gradient: 'from-pink-50 to-white dark:from-pink-900/20 dark:to-slate-900'
            };
        case NodeType.BYTES_TO_JSON:
            return {
                icon: Box,
                color: 'text-cyan-500',
                bg: 'bg-cyan-500',
                border: 'border-cyan-200 dark:border-cyan-900',
                gradient: 'from-cyan-50 to-white dark:from-cyan-900/20 dark:to-slate-900'
            };
        case NodeType.FILE_TO_BYTES:
            return {
                icon: FileDown,
                color: 'text-green-500',
                bg: 'bg-green-500',
                border: 'border-green-200 dark:border-green-900',
                gradient: 'from-green-50 to-white dark:from-green-900/20 dark:to-slate-900'
            };
        case NodeType.BYTES_TO_FILE:
            return {
                icon: FileUp,
                color: 'text-teal-500',
                bg: 'bg-teal-500',
                border: 'border-teal-200 dark:border-teal-900',
                gradient: 'from-teal-50 to-white dark:from-teal-900/20 dark:to-slate-900'
            };
        case NodeType.CSV_TO_JSON:
            return {
                icon: Table,
                color: 'text-cyan-500',
                bg: 'bg-cyan-500',
                border: 'border-cyan-200 dark:border-cyan-900',
                gradient: 'from-cyan-50 to-white dark:from-cyan-900/20 dark:to-slate-900'
            };

        // Comment
        case NodeType.COMMENT:
            return {
                icon: MessageSquare,
                color: 'text-slate-400',
                bg: 'bg-slate-400',
                border: 'border-slate-200 dark:border-slate-700',
                gradient: 'from-slate-50 to-white dark:from-slate-800 dark:to-slate-900'
            };

        // Advanced workflow patterns
        case NodeType.SUB_WORKFLOW:
            return {
                icon: Network,
                color: 'text-indigo-500',
                bg: 'bg-indigo-500',
                border: 'border-indigo-200 dark:border-indigo-900',
                gradient: 'from-indigo-50 to-white dark:from-indigo-900/20 dark:to-slate-900'
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

                {/* Sub-workflow progress */}
                {(data.nodeType === NodeType.SUB_WORKFLOW || data.type === NodeType.SUB_WORKFLOW) && data.subWorkflowProgress && (
                    <SubWorkflowProgress
                        total={data.subWorkflowProgress.total}
                        completed={data.subWorkflowProgress.completed}
                        failed={data.subWorkflowProgress.failed}
                        running={data.subWorkflowProgress.running}
                    />
                )}

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
