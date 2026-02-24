import React from 'react';
import { HardDrive } from 'lucide-react';
import type { GoogleDriveNodeConfig as GoogleDriveNodeConfigType } from '@/types/nodeConfigs';
import { VariableAutocomplete } from '@/components/builder/VariableAutocomplete';
import { useTranslation } from '@/store/translations';

interface Props {
    config: GoogleDriveNodeConfigType;
    nodeId?: string;
    onChange: (config: GoogleDriveNodeConfigType) => void;
}

export const GoogleDriveNodeConfigComponent: React.FC<Props> = ({ config, onChange }) => {
    const t = useTranslation();

    // ALWAYS create safeConfig with defaults to prevent undefined errors
    const safeConfig: GoogleDriveNodeConfigType = {
        operation: config?.operation || 'list_files',
        credentials: config?.credentials || '',
        file_name: config?.file_name || '',
        folder_name: config?.folder_name || '',
        file_id: config?.file_id || '',
        parent_folder_id: config?.parent_folder_id || '',
        destination_folder_id: config?.destination_folder_id || '',
        max_results: config?.max_results ?? 100,
        order_by: config?.order_by || 'modifiedTime desc',
    };

    // Handlers call onChange directly with safeConfig spread
    const handleOperationChange = (value: "create_spreadsheet" | "create_folder" | "list_files" | "delete" | "move" | "copy") => {
        onChange({ ...safeConfig, operation: value });
    };

    const handleCredentialsChange = (value: string) => {
        onChange({ ...safeConfig, credentials: value });
    };

    const handleFileNameChange = (value: string) => {
        onChange({ ...safeConfig, file_name: value });
    };

    const handleFolderNameChange = (value: string) => {
        onChange({ ...safeConfig, folder_name: value });
    };

    const handleFileIdChange = (value: string) => {
        onChange({ ...safeConfig, file_id: value });
    };

    const handleParentFolderIdChange = (value: string) => {
        onChange({ ...safeConfig, parent_folder_id: value });
    };

    const handleDestinationFolderIdChange = (value: string) => {
        onChange({ ...safeConfig, destination_folder_id: value });
    };

    const handleMaxResultsChange = (value: string) => {
        const num = parseInt(value, 10);
        onChange({ ...safeConfig, max_results: isNaN(num) ? 100 : num });
    };

    const handleOrderByChange = (value: string) => {
        onChange({ ...safeConfig, order_by: value });
    };

    const showFileNameField = safeConfig.operation === 'create_spreadsheet' || safeConfig.operation === 'copy';
    const showFolderNameField = safeConfig.operation === 'create_folder';
    const showFileIdField = safeConfig.operation === 'delete' || safeConfig.operation === 'move' || safeConfig.operation === 'copy';
    const showDestinationFolderIdField = safeConfig.operation === 'move' || safeConfig.operation === 'copy';
    const showListOptions = safeConfig.operation === 'list_files';

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/10 dark:to-indigo-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-4 flex items-start gap-3">
                <HardDrive className="text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" size={18} />
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">{t.nodeConfig.googleDrive.title}</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        {t.nodeConfig.googleDrive.description}
                    </p>
                </div>
            </div>

            {/* Fields */}
            <div className="space-y-3">
                {/* Operation */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleDrive.operation} <span className="text-red-500">{t.nodeConfig.required}</span>
                    </span>
                    <select
                        value={safeConfig.operation}
                        onChange={(e) => handleOperationChange(e.target.value as any)}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                    >
                        <option value="create_spreadsheet">{t.nodeConfig.googleDrive.operationCreateSpreadsheet}</option>
                        <option value="create_folder">{t.nodeConfig.googleDrive.operationCreateFolder}</option>
                        <option value="list_files">{t.nodeConfig.googleDrive.operationListFiles}</option>
                        <option value="delete">{t.nodeConfig.googleDrive.operationDelete}</option>
                        <option value="move">{t.nodeConfig.googleDrive.operationMove}</option>
                        <option value="copy">{t.nodeConfig.googleDrive.operationCopy}</option>
                    </select>
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleDrive.operationHint}
                    </span>
                </label>

                {/* Credentials */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleDrive.credentials} <span className="text-red-500">{t.nodeConfig.required}</span>
                    </span>
                    <textarea
                        value={safeConfig.credentials}
                        onChange={(e) => handleCredentialsChange(e.target.value)}
                        placeholder={t.nodeConfig.googleDrive.credentialsPlaceholder}
                        rows={4}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-xs font-mono resize-y"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleDrive.credentialsHint}
                    </span>
                </label>

                {/* File Name (for create_spreadsheet and copy) */}
                {showFileNameField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            {t.nodeConfig.googleDrive.fileName} {safeConfig.operation === 'create_spreadsheet' && <span className="text-slate-400">{t.nodeConfig.googleDrive.fileNameOptional}</span>}
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.file_name || ''}
                            onChange={handleFileNameChange}
                            placeholder={safeConfig.operation === 'create_spreadsheet' ? t.nodeConfig.googleDrive.fileNamePlaceholder : 'Copy of Original File'}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {t.nodeConfig.googleDrive.fileNameHint}
                        </span>
                    </label>
                )}

                {/* Folder Name (for create_folder) */}
                {showFolderNameField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            {t.nodeConfig.googleDrive.folderName} <span className="text-slate-400">{t.nodeConfig.optional}</span>
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.folder_name || ''}
                            onChange={handleFolderNameChange}
                            placeholder={t.nodeConfig.googleDrive.folderNamePlaceholder}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {t.nodeConfig.googleDrive.folderNameHint}
                        </span>
                    </label>
                )}

                {/* File ID (for delete, move, copy) */}
                {showFileIdField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            {t.nodeConfig.googleDrive.fileId} <span className="text-red-500">{t.nodeConfig.required}</span>
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.file_id || ''}
                            onChange={handleFileIdChange}
                            placeholder={t.nodeConfig.googleDrive.fileIdPlaceholder}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm font-mono"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {t.nodeConfig.googleDrive.fileIdHint}
                        </span>
                    </label>
                )}

                {/* Parent Folder ID (optional for create operations and list) */}
                {(safeConfig.operation === 'create_spreadsheet' || safeConfig.operation === 'create_folder' || safeConfig.operation === 'list_files') && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            {t.nodeConfig.googleDrive.parentFolderId} <span className="text-slate-400">{t.nodeConfig.optional}</span>
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.parent_folder_id || ''}
                            onChange={handleParentFolderIdChange}
                            placeholder={t.nodeConfig.googleDrive.parentFolderIdPlaceholder}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm font-mono"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {t.nodeConfig.googleDrive.parentFolderIdHint}
                        </span>
                    </label>
                )}

                {/* Destination Folder ID (for move and copy) */}
                {showDestinationFolderIdField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            {t.nodeConfig.googleDrive.destinationFolderId} {safeConfig.operation === 'move' ? <span className="text-red-500">{t.nodeConfig.required}</span> : <span className="text-slate-400">{t.nodeConfig.optional}</span>}
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.destination_folder_id || ''}
                            onChange={handleDestinationFolderIdChange}
                            placeholder={t.nodeConfig.googleDrive.destinationFolderIdPlaceholder}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm font-mono"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {t.nodeConfig.googleDrive.destinationFolderIdHint}
                        </span>
                    </label>
                )}

                {/* List Options */}
                {showListOptions && (
                    <>
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                {t.nodeConfig.googleDrive.maxResults}
                            </span>
                            <input
                                type="number"
                                value={safeConfig.max_results || 100}
                                onChange={(e) => handleMaxResultsChange(e.target.value)}
                                min="1"
                                max="1000"
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                            />
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                {t.nodeConfig.googleDrive.maxResultsHint}
                            </span>
                        </label>

                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                {t.nodeConfig.googleDrive.orderBy}
                            </span>
                            <select
                                value={safeConfig.order_by}
                                onChange={(e) => handleOrderByChange(e.target.value)}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                            >
                                <option value="modifiedTime desc">{t.nodeConfig.googleDrive.orderByModifiedDesc}</option>
                                <option value="modifiedTime">{t.nodeConfig.googleDrive.orderByModified}</option>
                                <option value="createdTime desc">{t.nodeConfig.googleDrive.orderByCreatedDesc}</option>
                                <option value="createdTime">{t.nodeConfig.googleDrive.orderByCreated}</option>
                                <option value="name">{t.nodeConfig.googleDrive.orderByName}</option>
                                <option value="name desc">{t.nodeConfig.googleDrive.orderByNameDesc}</option>
                            </select>
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                {t.nodeConfig.googleDrive.orderByHint}
                            </span>
                        </label>
                    </>
                )}

                {/* Usage Hint */}
                <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <h4 className="text-xs font-semibold text-blue-900 dark:text-blue-300 mb-1">
                        {t.nodeConfig.googleDrive.outputVariables}
                    </h4>
                    <p className="text-xs text-blue-700 dark:text-blue-400">
                        {safeConfig.operation === 'list_files' && 'Returns: success, files (array), file_count'}
                        {safeConfig.operation === 'create_spreadsheet' && 'Returns: success, file_id, file_name, web_view_url'}
                        {safeConfig.operation === 'create_folder' && 'Returns: success, folder_id, file_name, web_view_url'}
                        {safeConfig.operation === 'delete' && 'Returns: success, file_id, file_name'}
                        {safeConfig.operation === 'move' && 'Returns: success, file_id, file_name, web_view_url'}
                        {safeConfig.operation === 'copy' && 'Returns: success, file_id, file_name, source_file_id'}
                    </p>
                </div>
            </div>
        </div>
    );
};

export default GoogleDriveNodeConfigComponent;
