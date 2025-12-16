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
import { useTranslation } from '@/store/translations';

interface FileStorageNodeConfigProps {
    config: FileStorageNodeConfig;
    nodeId?: string;
    onChange: (config: FileStorageNodeConfig) => void;
}

export const FileStorageNodeConfigComponent: React.FC<FileStorageNodeConfigProps> = ({
                                                                                         config,
                                                                                         nodeId,
                                                                                         onChange,
                                                                                     }) => {
    const t = useTranslation();

    const actionOptions = [
        {label: t.nodeConfig.fileStorage.actionStore, value: 'store'},
        {label: t.nodeConfig.fileStorage.actionGet, value: 'get'},
        {label: t.nodeConfig.fileStorage.actionDelete, value: 'delete'},
        {label: t.nodeConfig.fileStorage.actionList, value: 'list'},
        {label: t.nodeConfig.fileStorage.actionMetadata, value: 'metadata'},
    ] as const;

    const fileSourceOptions = [
        {label: t.nodeConfig.fileStorage.sourceTypeUrl, value: 'url'},
        {label: t.nodeConfig.fileStorage.sourceTypeBase64, value: 'base64'},
    ] as const;

    const accessScopeOptions = [
        {label: t.nodeConfig.fileStorage.accessScopeWorkflow, value: 'workflow'},
        {label: t.nodeConfig.fileStorage.accessScopeEdge, value: 'edge'},
        {label: t.nodeConfig.fileStorage.accessScopeResult, value: 'result'},
    ] as const;
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
                    {t.nodeConfig.fileStorage.action}
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
                    {t.nodeConfig.fileStorage.storageId}
                </label>
                <VariableAutocomplete
                    value={localConfig.storage_id || ''}
                    onChange={(value) => updateConfig({storage_id: value})}
                    placeholder={t.nodeConfig.fileStorage.storageIdPlaceholder}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400">
                    {t.nodeConfig.fileStorage.storageIdHint}
                </p>
            </div>

            {/* Store Action Fields */}
            {localConfig.action === 'store' && (
                <>
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 space-y-4">
                        <h5 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                            {t.nodeConfig.fileStorage.fileSource}
                        </h5>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.sourceType}
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
                                {localConfig.file_source === 'url' ? t.nodeConfig.fileStorage.fileUrl : t.nodeConfig.fileStorage.base64Data}
                            </label>
                            {localConfig.file_source === 'url' ? (
                                <VariableAutocomplete
                                    value={localConfig.file_url || ''}
                                    onChange={(value) => updateConfig({file_url: value})}
                                    placeholder={t.nodeConfig.fileStorage.fileUrlPlaceholder}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                                />
                            ) : (
                                <VariableAutocomplete
                                    value={localConfig.file_data || ''}
                                    onChange={(value) => updateConfig({file_data: value})}
                                    placeholder={t.nodeConfig.fileStorage.base64Placeholder}
                                    type="textarea"
                                    rows={3}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100 resize-y"
                                />
                            )}
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.fileName}
                            </label>
                            <VariableAutocomplete
                                value={localConfig.file_name || ''}
                                onChange={(value) => updateConfig({file_name: value})}
                                placeholder={t.nodeConfig.fileStorage.fileNamePlaceholder}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.mimeType}
                            </label>
                            <VariableAutocomplete
                                value={localConfig.mime_type || ''}
                                onChange={(value) => updateConfig({mime_type: value})}
                                placeholder={t.nodeConfig.fileStorage.mimeTypePlaceholder}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>
                    </div>

                    {/* Access Scope & Options */}
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 space-y-4">
                        <h5 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                            {t.nodeConfig.fileStorage.storageOptions}
                        </h5>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.accessScope}
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
                                {t.nodeConfig.fileStorage.ttl}
                            </label>
                            <VariableAutocomplete
                                value={String(localConfig.ttl || 0)}
                                onChange={(value) => updateConfig({ttl: Number(value) || 0})}
                                placeholder={t.nodeConfig.fileStorage.ttlPlaceholder}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.tags}
                            </label>
                            <VariableAutocomplete
                                value={tagsStr}
                                onChange={handleTagsChange}
                                placeholder={t.nodeConfig.fileStorage.tagsPlaceholder}
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
                        {t.nodeConfig.fileStorage.fileId}
                    </label>
                    <VariableAutocomplete
                        value={localConfig.file_id || ''}
                        onChange={(value) => updateConfig({file_id: value})}
                        placeholder={t.nodeConfig.fileStorage.fileIdPlaceholder}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                    />
                </div>
            )}

            {/* List Action Fields */}
            {localConfig.action === 'list' && (
                <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 space-y-4">
                    <h5 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                        {t.nodeConfig.fileStorage.filters}
                    </h5>

                    <div className="flex flex-col gap-1.5">
                        <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                            {t.nodeConfig.fileStorage.accessScope}
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
                            <option value="">{t.nodeConfig.fileStorage.allScopes}</option>
                            {accessScopeOptions.map((option) => (
                                <option key={option.value} value={option.value}>
                                    {option.label}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div className="flex flex-col gap-1.5">
                        <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                            {t.nodeConfig.fileStorage.tagsFilter}
                        </label>
                        <VariableAutocomplete
                            value={tagsStr}
                            onChange={handleTagsChange}
                            placeholder={t.nodeConfig.fileStorage.tagsPlaceholder}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.limit}
                            </label>
                            <VariableAutocomplete
                                value={String(localConfig.limit || 100)}
                                onChange={(value) => updateConfig({limit: Number(value) || 100})}
                                placeholder={t.nodeConfig.fileStorage.limitPlaceholder}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                            />
                        </div>
                        <div className="flex flex-col gap-1.5">
                            <label className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                {t.nodeConfig.fileStorage.offset}
                            </label>
                            <VariableAutocomplete
                                value={String(localConfig.offset || 0)}
                                onChange={(value) => updateConfig({offset: Number(value) || 0})}
                                placeholder={t.nodeConfig.fileStorage.offsetPlaceholder}
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
