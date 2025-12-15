import React, {useEffect, useState} from 'react';
import {useDagStore} from '@/store/dagStore.ts';
import {useTranslation} from '@/store/translations.ts';
import {Trash2, X} from 'lucide-react';
import {NodeType} from '@/types.ts';
import {VariableAutocomplete} from '@/components';
import {Button} from '../ui';

// Import all node config components
import {HTTPNodeConfigComponent} from '@/components/nodes/config/HTTPNodeConfig.tsx';
import {LLMNodeConfigComponent} from '@/components/nodes/config/LLMNodeConfig.tsx';
import {TelegramNodeConfigComponent} from '@/components/nodes/config/TelegramNodeConfig.tsx';
import {TelegramDownloadNodeConfig} from '@/components/nodes/config';
import {TelegramParseNodeConfigComponent} from '@/components/nodes/config/TelegramParseNodeConfig.tsx';
import {TelegramCallbackNodeConfigComponent} from '@/components/nodes/config/TelegramCallbackNodeConfig.tsx';
import {ConditionalNodeConfigComponent} from '@/components/nodes/config/ConditionalNodeConfig.tsx';
import {MergeNodeConfigComponent} from '@/components/nodes/config/MergeNodeConfig.tsx';
import {DelayNodeConfig} from '@/components/nodes/config/DelayNodeConfig.tsx';
import {TransformNodeConfigComponent} from '@/components/nodes/config/TransformNodeConfig.tsx';
import {FunctionCallNodeConfigComponent} from '@/components/nodes/config/FunctionCallNodeConfig.tsx';
import {FileStorageNodeConfigComponent} from '@/components/nodes/config/FileStorageNodeConfig.tsx';
import {
    Base64ToBytesNodeConfig,
    BytesToBase64NodeConfig,
    BytesToFileNodeConfig,
    BytesToJsonNodeConfig,
    FileToBytesNodeConfig,
    JsonToStringNodeConfig,
    StringToJsonNodeConfig,
} from '@/components/nodes/config';
import {HTMLCleanNodeConfigComponent} from '../nodes/config/HTMLCleanNodeConfig';
import {RSSParserNodeConfigComponent} from '../nodes/config/RSSParserNodeConfig';
import {CSVToJSONNodeConfigComponent} from '../nodes/config/CSVToJSONNodeConfig';

// Import default configs
import {DEFAULT_NODE_CONFIGS} from '@/types/nodeConfigs';

export const PropertiesPanel: React.FC = () => {
    const {selectedNodeId, nodes, updateNodeData, setSelectedNodeId, deleteNode} = useDagStore();
    const t = useTranslation();

    const selectedNode = nodes.find((n) => n.id === selectedNodeId);
    const [formData, setFormData] = useState<Record<string, any>>({});

    useEffect(() => {
        if (selectedNode) {
            setFormData({
                label: selectedNode.data.label || '',
                description: selectedNode.data.description || '',
                ...selectedNode.data.config || {},
            });
        }
    }, [selectedNode]);

    const isOpen = !!selectedNode;

    const handleChange = (field: string, value: any) => {
        if (!selectedNode) return;
        const newData = {...formData, [field]: value};
        setFormData(newData);

        // Auto update store for label/description
        if (field === 'label' || field === 'description') {
            updateNodeData(selectedNode.id, {
                ...selectedNode.data,
                [field]: value,
            });
        } else {
            updateNodeData(selectedNode.id, {
                ...selectedNode.data,
                config: {
                    ...selectedNode.data.config,
                    [field]: value
                }
            });
        }
    };

    const handleConfigChange = (newConfig: Record<string, any>) => {
        if (!selectedNode) return;
        updateNodeData(selectedNode.id, {
            ...selectedNode.data,
            config: newConfig,
        });
    };

    const handleDelete = () => {
        if (selectedNode && confirm(t.common.delete + '?')) {
            deleteNode(selectedNode.id);
        }
    };

    // Get current node config with defaults
    const getNodeConfig = () => {
        if (!selectedNode) return {};
        const nodeType = selectedNode.data.type as string;
        const defaultConfig = DEFAULT_NODE_CONFIGS[nodeType] || {};
        return {...defaultConfig, ...selectedNode.data.config};
    };

    return (
        <div
            className={`absolute right-0 top-0 h-full w-80 bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800 shadow-2xl z-20 transition-transform duration-300 ease-in-out transform ${
                isOpen ? 'translate-x-0' : 'translate-x-full'
            }`}
        >
            {selectedNode && (
                <div className="flex flex-col h-full">
                    <div
                        className="px-5 py-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between bg-slate-50/50 dark:bg-slate-900/50 backdrop-blur-sm">
                        <div>
                            <h2 className="font-bold text-slate-800 dark:text-slate-100">{t.builder.properties}</h2>
                            <p className="text-xs text-slate-500 dark:text-slate-400 font-mono mt-0.5">ID: {selectedNode.id}</p>
                        </div>
                        <Button
                            variant="ghost"
                            size="sm"
                            icon={<X size={18}/>}
                            onClick={() => setSelectedNodeId(null)}
                        />
                    </div>

                    <div className="flex-1 overflow-y-auto p-5 space-y-6">

                        {/* General Settings */}
                        <div className="space-y-4">
                            <h3 className="text-xs font-bold text-slate-900 dark:text-slate-100 uppercase tracking-wider">{t.builder.general}</h3>

                            <div className="space-y-1.5">
                                <label
                                    className="text-xs font-semibold text-slate-600 dark:text-slate-400">{t.builder.name}</label>
                                <input
                                    type="text"
                                    value={formData.label || ''}
                                    onChange={(e) => handleChange('label', e.target.value)}
                                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200 placeholder-slate-400"
                                />
                            </div>

                            <div className="space-y-1.5">
                                <label
                                    className="text-xs font-semibold text-slate-600 dark:text-slate-400">{t.builder.description}</label>
                                <VariableAutocomplete
                                    type="textarea"
                                    value={formData.description || ''}
                                    onChange={(val) => handleChange('description', val)}
                                    rows={3}
                                    placeholder={t.builder.description + "..."}
                                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-none text-slate-800 dark:text-slate-200 placeholder-slate-400"
                                />
                            </div>
                        </div>

                        {/* Dynamic Fields based on Node Type */}
                        <div className="space-y-4 pt-4 border-t border-slate-100 dark:border-slate-800">
                            <h3 className="text-xs font-bold text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                {t.builder.nodeConfig}
                            </h3>

                            {renderNodeConfigComponent(
                                selectedNode.data.type as NodeType,
                                getNodeConfig(),
                                selectedNode.id,
                                handleConfigChange
                            )}
                        </div>

                    </div>

                    <div
                        className="p-4 border-t border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50">
                        <Button
                            variant="danger"
                            fullWidth
                            icon={<Trash2 size={16}/>}
                            onClick={handleDelete}
                        >
                            {t.builder.deleteNode}
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
};

/**
 * Render the appropriate config component based on node type.
 * NodeType enum values ARE the backend type strings (e.g., NodeType.API_REQUEST = 'http'),
 * so we only need to match against NodeType enum members.
 */
const renderNodeConfigComponent = (
    type: NodeType | string,
    config: Record<string, any>,
    nodeId: string,
    onChange: (config: Record<string, any>) => void
) => {
    switch (type) {
        // HTTP Request (value: 'http')
        case NodeType.HTTP:
            return (
                <HTTPNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // LLM / AI (value: 'llm')
        case NodeType.LLM:
            return (
                <LLMNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Transform (value: 'transform')
        case NodeType.TRANSFORM:
            return (
                <TransformNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Function Call (value: 'function_call')
        case NodeType.FUNCTION_CALL:
            return (
                <FunctionCallNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Telegram Send (value: 'telegram')
        case NodeType.TELEGRAM:
            return (
                <TelegramNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Telegram Download (value: 'telegram_download')
        case NodeType.TELEGRAM_DOWNLOAD:
            return (
                <TelegramDownloadNodeConfig
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Telegram Parse (value: 'telegram_parse')
        case NodeType.TELEGRAM_PARSE:
            return (
                <TelegramParseNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Telegram Callback (value: 'telegram_callback')
        case NodeType.TELEGRAM_CALLBACK:
            return (
                <TelegramCallbackNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Conditional (value: 'conditional')
        case NodeType.CONDITIONAL:
            return (
                <ConditionalNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Merge (value: 'merge')
        case NodeType.MERGE:
            return (
                <MergeNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Delay / Scheduler (value: 'delay')
        case NodeType.DELAY:
            return (
                <DelayNodeConfig
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // File Storage (value: 'file_storage')
        case NodeType.FILE_STORAGE:
            return (
                <FileStorageNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: Base64 to Bytes (value: 'base64_to_bytes')
        case NodeType.BASE64_TO_BYTES:
            return (
                <Base64ToBytesNodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: Bytes to Base64 (value: 'bytes_to_base64')
        case NodeType.BYTES_TO_BASE64:
            return (
                <BytesToBase64NodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: String to JSON (value: 'string_to_json')
        case NodeType.STRING_TO_JSON:
            return (
                <StringToJsonNodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: JSON to String (value: 'json_to_string')
        case NodeType.JSON_TO_STRING:
            return (
                <JsonToStringNodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: Bytes to JSON (value: 'bytes_to_json')
        case NodeType.BYTES_TO_JSON:
            return (
                <BytesToJsonNodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: File to Bytes (value: 'file_to_bytes')
        case NodeType.FILE_TO_BYTES:
            return (
                <FileToBytesNodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Adapter: Bytes to File (value: 'bytes_to_file')
        case NodeType.BYTES_TO_FILE:
            return (
                <BytesToFileNodeConfig
                    config={config}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // HTML Clean (value: 'html_clean')
        case NodeType.HTML_CLEAN:
            return (
                <HTMLCleanNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // RSS Parser (value: 'rss_parser')
        case NodeType.RSS_PARSER:
            return (
                <RSSParserNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // CSV to JSON (value: 'csv_to_json')
        case NodeType.CSV_TO_JSON:
            return (
                <CSVToJSONNodeConfigComponent
                    config={config as any}
                    nodeId={nodeId}
                    onChange={onChange}
                />
            );

        // Default - no config
        default:
            return (
                <p className="text-sm text-slate-400 italic">
                    No specific configuration available for this node type.
                </p>
            );
    }
};
