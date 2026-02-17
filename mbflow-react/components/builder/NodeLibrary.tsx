import React, {useMemo, useState} from 'react';
import {
    Box,
    Braces, BrushCleaning,
    CheckCircle,
    Clock,
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
    GripVertical,
    Lock,
    Network,
    Search,
    Send,
    Sheet,
    Sparkles,
    Table,
    Unlock,
    X,
    Zap,
    Rss,
} from 'lucide-react';
import {NodeType} from '@/types.ts';
import {useTranslation} from '@/store/translations.ts';
import {useUIStore} from '@/store/uiStore.ts';
import {Button} from '@/components/ui';

// Node definitions with metadata
interface NodeDefinition {
    type: NodeType;
    labelKey: string;  // Key in t.nodes translation object
    icon: React.ReactNode;
    category: 'triggers' | 'actions' | 'telegram' | 'logic' | 'storage' | 'adapters';
}

const nodeDefinitions: NodeDefinition[] = [
    // Triggers
    {
        type: NodeType.DELAY,
        labelKey: 'delay',
        icon: <Clock size={16} className="text-cyan-500"/>,
        category: 'triggers'
    },

    // Actions
    {
        type: NodeType.HTTP,
        labelKey: 'http',
        icon: <Globe size={16} className="text-green-500"/>,
        category: 'actions'
    },
    {
        type: NodeType.RSS_PARSER,
        labelKey: 'RSS',
        icon: <Rss size={16} className="text-orange-500"/>,
        category: 'actions'
    },
    {
        type: NodeType.LLM,
        labelKey: 'llm',
        icon: <Sparkles size={16} className="text-purple-500"/>,
        category: 'actions'
    },
    {
        type: NodeType.TRANSFORM,
        labelKey: 'transform',
        icon: <Zap size={16} className="text-amber-500"/>,
        category: 'actions'
    },
    {
        type: NodeType.FUNCTION_CALL,
        labelKey: 'functionCall',
        icon: <Code size={16} className="text-blue-500"/>,
        category: 'actions'
    },
    {
        type: NodeType.GOOGLE_SHEETS,
        labelKey: 'googleSheets',
        icon: <Sheet size={16} className="text-green-600"/>,
        category: 'actions'
    },
    {
        type: NodeType.GOOGLE_DRIVE,
        labelKey: 'googleDrive',
        icon: <Folder size={16} className="text-blue-600"/>,
        category: 'actions'
    },

    // Telegram
    {
        type: NodeType.TELEGRAM,
        labelKey: 'telegram',
        icon: <Send size={16} className="text-sky-500"/>,
        category: 'telegram'
    },
    {
        type: NodeType.TELEGRAM_DOWNLOAD,
        labelKey: 'telegramDownload',
        icon: <Download size={16} className="text-sky-500"/>,
        category: 'telegram'
    },
    {
        type: NodeType.TELEGRAM_PARSE,
        labelKey: 'telegramParse',
        icon: <FileSearch size={16} className="text-sky-500"/>,
        category: 'telegram'
    },
    {
        type: NodeType.TELEGRAM_CALLBACK,
        labelKey: 'telegramCallback',
        icon: <CheckCircle size={16} className="text-sky-500"/>,
        category: 'telegram'
    },

    // Logic
    {
        type: NodeType.CONDITIONAL,
        labelKey: 'conditional',
        icon: <GitBranch size={16} className="text-pink-500"/>,
        category: 'logic'
    },
    {
        type: NodeType.MERGE,
        labelKey: 'merge',
        icon: <GitMerge size={16} className="text-violet-500"/>,
        category: 'logic'
    },
    {
        type: NodeType.SUB_WORKFLOW,
        labelKey: 'subWorkflow',
        icon: <Network size={16} className="text-indigo-500"/>,
        category: 'logic'
    },

    // Storage
    {
        type: NodeType.FILE_STORAGE,
        labelKey: 'fileStorage',
        icon: <Folder size={16} className="text-teal-500"/>,
        category: 'storage'
    },

    // Adapters
    {
        type: NodeType.HTML_CLEAN,
        labelKey: 'HTMLCleaner',
        icon: <BrushCleaning size={16} className="text-orange-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.BASE64_TO_BYTES,
        labelKey: 'base64ToBytes',
        icon: <Unlock size={16} className="text-red-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.BYTES_TO_BASE64,
        labelKey: 'bytesToBase64',
        icon: <Lock size={16} className="text-amber-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.STRING_TO_JSON,
        labelKey: 'stringToJson',
        icon: <Braces size={16} className="text-violet-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.JSON_TO_STRING,
        labelKey: 'jsonToString',
        icon: <FileText size={16} className="text-pink-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.BYTES_TO_JSON,
        labelKey: 'bytesToJson',
        icon: <Box size={16} className="text-cyan-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.FILE_TO_BYTES,
        labelKey: 'fileToBytes',
        icon: <FileDown size={16} className="text-green-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.BYTES_TO_FILE,
        labelKey: 'bytesToFile',
        icon: <FileUp size={16} className="text-teal-500"/>,
        category: 'adapters'
    },
    {
        type: NodeType.CSV_TO_JSON,
        labelKey: 'csvToJson',
        icon: <Table size={16} className="text-cyan-500"/>,
        category: 'adapters'
    },
];

export const NodeLibrary: React.FC = () => {
    const t = useTranslation();
    const {isNodeLibraryOpen, toggleNodeLibrary} = useUIStore();
    const [searchQuery, setSearchQuery] = useState('');

    const onDragStart = (event: React.DragEvent, nodeType: NodeType) => {
        event.dataTransfer.setData('application/reactflow', nodeType);
        event.dataTransfer.effectAllowed = 'move';
    };

    // Filter nodes by search query
    const filteredNodes = useMemo(() => {
        if (!searchQuery.trim()) return nodeDefinitions;
        const query = searchQuery.toLowerCase();
        return nodeDefinitions.filter(node => {
            const nodesDict = t.nodes as Record<string, string>;
            const label = nodesDict[node.labelKey] || node.labelKey;
            return label.toLowerCase().includes(query);
        });
    }, [searchQuery, t.nodes]);

    // Group nodes by category
    const groupedNodes = useMemo(() => {
        const groups: Record<string, NodeDefinition[]> = {
            triggers: [],
            actions: [],
            telegram: [],
            logic: [],
            storage: [],
            adapters: [],
        };
        filteredNodes.forEach(node => {
            groups[node.category].push(node);
        });
        return groups;
    }, [filteredNodes]);

    const categoryLabels: Record<string, string> = {
        triggers: t.builder.triggers,
        actions: t.builder.actions,
        telegram: t.builder.telegram || 'TELEGRAM',
        logic: t.builder.logic,
        storage: t.builder.storage || 'STORAGE',
        adapters: t.builder.adapters || 'ADAPTERS',
    };

    return (
        <div
            className={`absolute left-0 top-0 h-full w-64 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col transition-transform duration-300 ease-in-out z-20 shadow-xl ${
                isNodeLibraryOpen ? 'translate-x-0' : '-translate-x-full'
            }`}
        >
            <div
                className="p-4 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 flex flex-col gap-3">
                <div className="flex items-center justify-between">
                    <h2 className="text-xs font-bold text-slate-900 dark:text-slate-100 uppercase tracking-wider flex items-center">
                        {t.builder.components}
                    </h2>
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={<X size={16}/>}
                        onClick={toggleNodeLibrary}
                    />
                </div>

                <div className="relative group">
                    <Search
                        className="absolute left-2.5 top-2.5 w-4 h-4 text-slate-400 group-focus-within:text-blue-500 transition-colors"/>
                    <input
                        type="text"
                        placeholder={t.common.search}
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="w-full pl-9 pr-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-700 dark:text-slate-200 placeholder-slate-400 transition-all shadow-sm"
                    />
                </div>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-5">
                {Object.entries(groupedNodes).map(([category, nodes]) => {
                    if (nodes.length === 0) return null;
                    return (
                        <div key={category}>
                            <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-2 ml-1">
                                {categoryLabels[category]}
                            </h3>
                            <div className="space-y-1.5">
                                {nodes.map(node => {
                                    const nodesDict = t.nodes as Record<string, string>;
                                    return (
                                        <DraggableNode
                                            key={node.type}
                                            type={node.type}
                                            label={nodesDict[node.labelKey] || node.labelKey}
                                            icon={node.icon}
                                            onDragStart={onDragStart}
                                        />
                                    );
                                })}
                            </div>
                        </div>
                    );
                })}
            </div>

            <div
                className="p-3 border-t border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 text-[10px] text-slate-400 text-center">
                {t.builder.dragToAdd}
            </div>
        </div>
    );
};

interface DraggableNodeProps {
    type: NodeType;
    label: string;
    icon: React.ReactNode;
    onDragStart: (event: React.DragEvent, type: NodeType) => void;
}

const DraggableNode: React.FC<DraggableNodeProps> = ({type, label, icon, onDragStart}) => {
    return (
        <div
            className="flex items-center p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg cursor-grab hover:border-blue-400 dark:hover:border-blue-500 hover:shadow-sm transition-all active:cursor-grabbing group select-none"
            draggable
            onDragStart={(event) => onDragStart(event, type)}
        >
            <div
                className="mr-3 p-1.5 bg-slate-50 dark:bg-slate-900 rounded-md border border-slate-100 dark:border-slate-800 group-hover:border-blue-200 dark:group-hover:border-blue-900 transition-colors">
                {icon}
            </div>
            <span className="text-sm font-medium text-slate-700 dark:text-slate-200 flex-1">{label}</span>
            <GripVertical size={14}
                          className="text-slate-300 dark:text-slate-600 opacity-0 group-hover:opacity-100 transition-opacity"/>
        </div>
    );
};