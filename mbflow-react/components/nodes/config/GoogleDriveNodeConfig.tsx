import React from 'react';
import { HardDrive } from 'lucide-react';
import type { GoogleDriveNodeConfig as GoogleDriveNodeConfigType } from '../../../types/nodeConfigs';
import { VariableAutocomplete } from '../../builder/VariableAutocomplete';

interface Props {
    config: GoogleDriveNodeConfigType;
    nodeId?: string;
    onChange: (config: GoogleDriveNodeConfigType) => void;
}

export const GoogleDriveNodeConfigComponent: React.FC<Props> = ({ config, onChange }) => {
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
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">Google Drive</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        Manage files and folders in Google Drive using service account credentials
                    </p>
                </div>
            </div>

            {/* Fields */}
            <div className="space-y-3">
                {/* Operation */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Operation <span className="text-red-500">*</span>
                    </span>
                    <select
                        value={safeConfig.operation}
                        onChange={(e) => handleOperationChange(e.target.value as any)}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                    >
                        <option value="create_spreadsheet">Create Spreadsheet</option>
                        <option value="create_folder">Create Folder</option>
                        <option value="list_files">List Files</option>
                        <option value="delete">Delete File/Folder</option>
                        <option value="move">Move File</option>
                        <option value="copy">Copy File</option>
                    </select>
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        Select the operation to perform on Google Drive
                    </span>
                </label>

                {/* Credentials */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Service Account Credentials (JSON) <span className="text-red-500">*</span>
                    </span>
                    <textarea
                        value={safeConfig.credentials}
                        onChange={(e) => handleCredentialsChange(e.target.value)}
                        placeholder='{"type":"service_account","project_id":"...","private_key":"...","client_email":"..."}'
                        rows={4}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-xs font-mono resize-y"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        Full JSON credentials from Google Cloud Console service account
                    </span>
                </label>

                {/* File Name (for create_spreadsheet and copy) */}
                {showFileNameField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            File Name {safeConfig.operation === 'create_spreadsheet' && <span className="text-slate-400">(optional)</span>}
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.file_name || ''}
                            onChange={handleFileNameChange}
                            placeholder={safeConfig.operation === 'create_spreadsheet' ? 'Untitled Spreadsheet' : 'Copy of Original File'}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {safeConfig.operation === 'create_spreadsheet' 
                                ? 'Name for the new spreadsheet (default: Untitled Spreadsheet)'
                                : 'Optional new name for copied file'}
                        </span>
                    </label>
                )}

                {/* Folder Name (for create_folder) */}
                {showFolderNameField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            Folder Name <span className="text-slate-400">(optional)</span>
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.folder_name || ''}
                            onChange={handleFolderNameChange}
                            placeholder="Untitled Folder"
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            Name for the new folder (default: Untitled Folder)
                        </span>
                    </label>
                )}

                {/* File ID (for delete, move, copy) */}
                {showFileIdField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            File ID <span className="text-red-500">*</span>
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.file_id || ''}
                            onChange={handleFileIdChange}
                            placeholder="1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm font-mono"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            Google Drive file ID to {safeConfig.operation}
                        </span>
                    </label>
                )}

                {/* Parent Folder ID (optional for create operations and list) */}
                {(safeConfig.operation === 'create_spreadsheet' || safeConfig.operation === 'create_folder' || safeConfig.operation === 'list_files') && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            Parent Folder ID <span className="text-slate-400">(optional)</span>
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.parent_folder_id || ''}
                            onChange={handleParentFolderIdChange}
                            placeholder="Leave empty for root folder"
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm font-mono"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            {safeConfig.operation === 'list_files' 
                                ? 'Folder to list files from (leave empty for root)'
                                : 'Parent folder where file/folder will be created (leave empty for root)'}
                        </span>
                    </label>
                )}

                {/* Destination Folder ID (for move and copy) */}
                {showDestinationFolderIdField && (
                    <label className="block">
                        <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                            Destination Folder ID {safeConfig.operation === 'move' ? <span className="text-red-500">*</span> : <span className="text-slate-400">(optional)</span>}
                        </span>
                        <VariableAutocomplete
                            value={safeConfig.destination_folder_id || ''}
                            onChange={handleDestinationFolderIdChange}
                            placeholder={safeConfig.operation === 'copy' ? 'Leave empty for same folder' : 'Destination folder ID'}
                            className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm font-mono"
                        />
                        <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                            Folder ID where file will be {safeConfig.operation === 'move' ? 'moved' : 'copied'}
                        </span>
                    </label>
                )}

                {/* List Options */}
                {showListOptions && (
                    <>
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                Max Results
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
                                Maximum number of files to return (default: 100)
                            </span>
                        </label>

                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                Order By
                            </span>
                            <select
                                value={safeConfig.order_by}
                                onChange={(e) => handleOrderByChange(e.target.value)}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                            >
                                <option value="modifiedTime desc">Modified Time (newest first)</option>
                                <option value="modifiedTime">Modified Time (oldest first)</option>
                                <option value="createdTime desc">Created Time (newest first)</option>
                                <option value="createdTime">Created Time (oldest first)</option>
                                <option value="name">Name (A-Z)</option>
                                <option value="name desc">Name (Z-A)</option>
                            </select>
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                Sort order for listed files
                            </span>
                        </label>
                    </>
                )}

                {/* Usage Hint */}
                <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <h4 className="text-xs font-semibold text-blue-900 dark:text-blue-300 mb-1">
                        Output Variables
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
