/**
 * FileStorageNodeConfig - React component for configuring File Storage nodes
 *
 * Ported from: /mbflow-ui/src/components/nodes/config/FileStorageNodeConfig.vue
 *
 * Features:
 * - Action selection (store, get, delete, list, metadata)
 * - Conditional fields based on action type
 * - File source selection (URL or Base64)
 * - Access scope and TTL configuration
 * - Tags management (comma-separated)
 * - List action with filters and pagination
 *
 * Usage:
 * ```tsx
 * <FileStorageNodeConfig
 *   config={fileStorageConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => console.log(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from 'react';
import {FileStorageNodeConfig} from '@/types/nodeConfigs.ts';
import {VariableAutocomplete} from '@/components';

interface FileStorageNodeConfigProps {
    config: FileStorageNodeConfig;
    nodeId?: string;
    onChange: (config: FileStorageNodeConfig) => void;
}

const actionOptions = [
    {label: 'Store File', value: 'store'},
    {label: 'Get File', value: 'get'},
    {label: 'Delete File', value: 'delete'},
    {label: 'List Files', value: 'list'},
    {label: 'Get Metadata', value: 'metadata'},
] as const;

const fileSourceOptions = [
    {label: 'URL', value: 'url'},
    {label: 'Base64 Data', value: 'base64'},
] as const;

const accessScopeOptions = [
    {label: 'Workflow', value: 'workflow'},
    {label: 'Edge (Connected Nodes)', value: 'edge'},
    {label: 'Result (Output Storage)', value: 'result'},
] as const;

export const FileStorageNodeConfigComponent: React.FC<FileStorageNodeConfigProps> = ({
                                                                                         config,
                                                                                         nodeId,
                                                                                         onChange,
                                                                                     }) => {
    const [localConfig, setLocalConfig] = useState<FileStorageNodeConfig>({
        action: config.action || 'store',
        storage_id: config.storage_id || '',
        file_source: config.file_source || 'url',
        file_data: config.file_data || '',
        file_url: config.file_url || '',
        file_name: config.file_name || '',
        mime_type: config.mime_type || '',
        file_id: config.file_id || '',
        access_scope: config.access_scope || 'workflow',
        ttl: config.ttl || 0,
        tags: config.tags || [],
        limit: config.limit || 100,
        offset: config.offset || 0,
    });

    const [tagsStr, setTagsStr] = useState<string>(
        config.tags?.join(', ') || ''
    );

    useEffect(() => {
        const newConfig = {
            action: config.action || 'store',
            storage_id: config.storage_id || '',
            file_source: config.file_source || 'url',
            file_data: config.file_data || '',
            file_url: config.file_url || '',
            file_name: config.file_name || '',
            mime_type: config.mime_type || '',
            file_id: config.file_id || '',
            access_scope: config.access_scope || 'workflow',
            ttl: config.ttl || 0,
            tags: config.tags || [],
            limit: config.limit || 100,
            offset: config.offset || 0,
        };

        if (JSON.stringify(newConfig) !== JSON.stringify(localConfig)) {
            setLocalConfig(newConfig);
            setTagsStr(newConfig.tags?.join(', ') || '');
        }
    }, [config]);

    useEffect(() => {
        const configToEmit = {...localConfig};

        if (tagsStr) {
            configToEmit.tags = tagsStr
                .split(',')
                .map((t) => t.trim())
                .filter((t) => t);
        } else {
            configToEmit.tags = [];
        }

        if (JSON.stringify(configToEmit) !== JSON.stringify(config)) {
            onChange(configToEmit);
        }
    }, [localConfig, tagsStr]);

    const updateConfig = (updates: Partial<FileStorageNodeConfig>) => {
        setLocalConfig((prev) => ({...prev, ...updates}));
    };

    const handleTagsChange = (value: string) => {
        setTagsStr(value);
    };

    return (
        <div className="flex flex-col gap-4">
            {/* Action Selection */}
            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                    Action
                </label>
                <select
                    value={localConfig.action}
                    onChange={(e) =>
                        updateConfig({
                            action: e.target.value as FileStorageNodeConfig['action'],
                        })
                    }
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                >
                    {actionOptions.map((option) => (
                        <option key={option.value} value={option.value}>
                            {option.label}
                        </option>
                    ))}
                </select>
            </div>

            {/* Storage ID (optional) */}
            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                    Storage ID (optional)
                </label>
                <VariableAutocomplete
                    value={localConfig.storage_id || ''}
                    onChange={(value) => updateConfig({storage_id: value})}
                    placeholder="default"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400">
                    Leave empty for default storage
                </p>
            </div>

            {/* Store Action Fields */}
            {localConfig.action === 'store' && (
                <>
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 space-y-4">
                        <h5 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                            File Source
                        </h5>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                Source Type
                            </label>
                            <select
                                value={localConfig.file_source}
                                onChange={(e) =>
                                    updateConfig({
                                        file_source: e.target.value as 'url' | 'base64',
                                    })
                                }
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                            >
                                {fileSourceOptions.map((option) => (
                                    <option key={option.value} value={option.value}>
                                        {option.label}
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {localConfig.file_source === 'url' ? 'File URL' : 'Base64 Data'}
                            </label>
                            {localConfig.file_source === 'url' ? (
                                <VariableAutocomplete
                                    value={localConfig.file_url || ''}
                                    onChange={(value) => updateConfig({file_url: value})}
                                    placeholder="https://example.com/document.pdf or {{input.url}}"
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                                />
                            ) : (
                                <VariableAutocomplete
                                    value={localConfig.file_data || ''}
                                    onChange={(value) => updateConfig({file_data: value})}
                                    placeholder="{{input.base64_data}}"
                                    type="textarea"
                                    rows={3}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100 resize-y"
                                />
                            )}
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                File Name
                            </label>
                            <VariableAutocomplete
                                value={localConfig.file_name || ''}
                                onChange={(value) => updateConfig({file_name: value})}
                                placeholder="document.pdf or {{input.filename}}"
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                MIME Type (optional)
                            </label>
                            <VariableAutocomplete
                                value={localConfig.mime_type || ''}
                                onChange={(value) => updateConfig({mime_type: value})}
                                placeholder="Auto-detected if empty"
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>
                    </div>

                    {/* Access Scope & Options */}
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 space-y-4">
                        <h5 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                            Storage Options
                        </h5>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                Access Scope
                            </label>
                            <select
                                value={localConfig.access_scope}
                                onChange={(e) =>
                                    updateConfig({
                                        access_scope: e.target.value as 'workflow' | 'edge' | 'result',
                                    })
                                }
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                            >
                                {accessScopeOptions.map((option) => (
                                    <option key={option.value} value={option.value}>
                                        {option.label}
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                TTL (seconds, 0 = no expiration)
                            </label>
                            <VariableAutocomplete
                                value={String(localConfig.ttl || 0)}
                                onChange={(value) => updateConfig({ttl: Number(value) || 0})}
                                placeholder="0"
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                Tags (comma-separated)
                            </label>
                            <VariableAutocomplete
                                value={tagsStr}
                                onChange={handleTagsChange}
                                placeholder="document, important"
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>
                    </div>
                </>
            )}

            {/* Get/Delete/Metadata Action Fields */}
            {['get', 'delete', 'metadata'].includes(localConfig.action) && (
                <div className="flex flex-col gap-1.5">
                    <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                        File ID
                    </label>
                    <VariableAutocomplete
                        value={localConfig.file_id || ''}
                        onChange={(value) => updateConfig({file_id: value})}
                        placeholder="{{input.file_id}}"
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                    />
                </div>
            )}

            {/* List Action Fields */}
            {localConfig.action === 'list' && (
                <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 space-y-4">
                    <h5 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                        Filters
                    </h5>

                    <div className="flex flex-col gap-1.5">
                        <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                            Access Scope (optional)
                        </label>
                        <select
                            value={localConfig.access_scope || ''}
                            onChange={(e) =>
                                updateConfig({
                                    access_scope: e.target.value as 'workflow' | 'edge' | 'result' | undefined,
                                })
                            }
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                        >
                            <option value="">All Scopes</option>
                            {accessScopeOptions.map((option) => (
                                <option key={option.value} value={option.value}>
                                    {option.label}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                            Tags Filter (comma-separated)
                        </label>
                        <VariableAutocomplete
                            value={tagsStr}
                            onChange={handleTagsChange}
                            placeholder="document, important"
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                Limit
                            </label>
                            <VariableAutocomplete
                                value={String(localConfig.limit || 100)}
                                onChange={(value) => updateConfig({limit: Number(value) || 100})}
                                placeholder="100"
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>
                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                Offset
                            </label>
                            <VariableAutocomplete
                                value={String(localConfig.offset || 0)}
                                onChange={(value) => updateConfig({offset: Number(value) || 0})}
                                placeholder="0"
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default FileStorageNodeConfigComponent;
