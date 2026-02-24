import React, {useEffect, useState} from 'react';
import {useDagStore} from '@/store/dagStore.ts';
import {useTranslation} from '@/store/translations.ts';
import {Trash2, X} from 'lucide-react';
import {NodeType} from '@/types.ts';
import {VariableAutocomplete} from '@/components';
import {Button, ConfirmModal} from '@/components/ui';

import {getNodeConfigComponent} from '@/components/nodes/config/nodeConfigRegistry';
import {DefaultNodeConfig} from '@/components/nodes/config/DefaultNodeConfig';
import {DEFAULT_NODE_CONFIGS} from '@/types/nodeConfigs';

export const PropertiesPanel: React.FC = () => {
    const {selectedNodeId, nodes, updateNodeData, setSelectedNodeId, deleteNode} = useDagStore();
    const t = useTranslation();

    const selectedNode = nodes.find((n) => n.id === selectedNodeId);
    const [formData, setFormData] = useState<Record<string, any>>({});
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

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
        if (selectedNode) {
            deleteNode(selectedNode.id);
            setShowDeleteConfirm(false);
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
                            onClick={() => setShowDeleteConfirm(true)}
                        >
                            {t.builder.deleteNode}
                        </Button>
                    </div>
                </div>
            )}

            {/* Delete Confirmation Modal */}
            <ConfirmModal
                isOpen={showDeleteConfirm}
                onClose={() => setShowDeleteConfirm(false)}
                onConfirm={handleDelete}
                title={t.builder.deleteNode}
                message="Are you sure you want to delete this node? This action cannot be undone."
                confirmText={t.common.delete}
                cancelText={t.common.cancel}
                variant="danger"
            />
        </div>
    );
};

const renderNodeConfigComponent = (
    type: NodeType | string,
    config: Record<string, any>,
    nodeId: string,
    onChange: (config: Record<string, any>) => void
) => {
    const ConfigComponent = getNodeConfigComponent(type) || DefaultNodeConfig;

    return (
        <ConfigComponent
            config={config}
            nodeId={nodeId}
            onChange={onChange}
        />
    );
};
