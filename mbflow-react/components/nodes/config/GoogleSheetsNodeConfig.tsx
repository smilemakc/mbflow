import React from 'react';
import { Sheet } from 'lucide-react';
import type { GoogleSheetsNodeConfig as GoogleSheetsNodeConfigType } from '../../../types/nodeConfigs';
import { VariableAutocomplete } from '../../builder/VariableAutocomplete';

interface Props {
    config: GoogleSheetsNodeConfigType;
    nodeId?: string;
    onChange: (config: GoogleSheetsNodeConfigType) => void;
}

export const GoogleSheetsNodeConfigComponent: React.FC<Props> = ({ config, onChange }) => {
    // ALWAYS create safeConfig with defaults to prevent undefined errors
    const safeConfig: GoogleSheetsNodeConfigType = {
        operation: config?.operation || 'read',
        spreadsheet_id: config?.spreadsheet_id || '',
        sheet_name: config?.sheet_name || '',
        range: config?.range || '',
        credentials: config?.credentials || '',
        value_input_option: config?.value_input_option || 'USER_ENTERED',
        major_dimension: config?.major_dimension || 'ROWS',
        columns: config?.columns || '',
    };

    // Handlers call onChange directly with safeConfig spread
    const handleOperationChange = (value: "read" | "write" | "append") => {
        onChange({ ...safeConfig, operation: value });
    };

    const handleSpreadsheetIdChange = (value: string) => {
        onChange({ ...safeConfig, spreadsheet_id: value });
    };

    const handleSheetNameChange = (value: string) => {
        onChange({ ...safeConfig, sheet_name: value });
    };

    const handleRangeChange = (value: string) => {
        onChange({ ...safeConfig, range: value });
    };

    const handleCredentialsChange = (value: string) => {
        onChange({ ...safeConfig, credentials: value });
    };

    const handleValueInputOptionChange = (value: "RAW" | "USER_ENTERED") => {
        onChange({ ...safeConfig, value_input_option: value });
    };

    const handleMajorDimensionChange = (value: "ROWS" | "COLUMNS") => {
        onChange({ ...safeConfig, major_dimension: value });
    };

    const handleColumnsChange = (value: string) => {
        onChange({ ...safeConfig, columns: value });
    };

    const showWriteOptions = safeConfig.operation === 'write' || safeConfig.operation === 'append';

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/10 dark:to-emerald-900/10 border border-green-200 dark:border-green-800 rounded-lg p-4 flex items-start gap-3">
                <Sheet className="text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" size={18} />
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">Google Sheets</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        Read, write, and append data to Google Sheets using service account credentials
                    </p>
                </div>
            </div>

            {/* Fields */}
            <div className="space-y-3">
                {/* Operation */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Operation
                    </span>
                    <select
                        value={safeConfig.operation}
                        onChange={(e) => handleOperationChange(e.target.value as "read" | "write" | "append")}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                    >
                        <option value="read">Read</option>
                        <option value="write">Write (overwrite)</option>
                        <option value="append">Append</option>
                    </select>
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        Choose whether to read, write (overwrite), or append data
                    </span>
                </label>

                {/* Spreadsheet ID */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Spreadsheet ID <span className="text-red-500">*</span>
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.spreadsheet_id}
                        onChange={handleSpreadsheetIdChange}
                        placeholder="1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm font-mono"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        The ID from the spreadsheet URL (between /d/ and /edit)
                    </span>
                </label>

                {/* Sheet Name */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Sheet Name
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.sheet_name}
                        onChange={handleSheetNameChange}
                        placeholder="Sheet1"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        Optional. The name of the sheet tab (defaults to first sheet)
                    </span>
                </label>

                {/* Range */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Range (A1 notation)
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.range}
                        onChange={handleRangeChange}
                        placeholder="A1:D10"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm font-mono"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        Optional. Cell range in A1 notation (e.g., A1:D10, B2:C5)
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
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-xs font-mono resize-y"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        Full JSON credentials from Google Cloud Console service account
                    </span>
                </label>

                {/* Write/Append Options */}
                {showWriteOptions && (
                    <>
                        {/* Value Input Option */}
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                Value Input Option
                            </span>
                            <select
                                value={safeConfig.value_input_option}
                                onChange={(e) => handleValueInputOptionChange(e.target.value as "RAW" | "USER_ENTERED")}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                            >
                                <option value="USER_ENTERED">User Entered (parse formulas, dates)</option>
                                <option value="RAW">Raw (store as-is)</option>
                            </select>
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                How input values should be interpreted
                            </span>
                        </label>

                        {/* Major Dimension */}
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                Major Dimension
                            </span>
                            <select
                                value={safeConfig.major_dimension}
                                onChange={(e) => handleMajorDimensionChange(e.target.value as "ROWS" | "COLUMNS")}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                            >
                                <option value="ROWS">Rows (default)</option>
                                <option value="COLUMNS">Columns</option>
                            </select>
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                How data arrays should be interpreted
                            </span>
                        </label>

                        {/* Columns Mapping */}
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                Column Mapping
                            </span>
                            <VariableAutocomplete
                                value={safeConfig.columns || ''}
                                onChange={handleColumnsChange}
                                placeholder="title, description, link, categories, pubDate"
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm font-mono"
                            />
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                Optional. Comma-separated list of field names to extract from objects in order.
                                If empty, all fields are extracted in alphabetical order.
                                Arrays/objects in values will be serialized as JSON strings.
                            </span>
                        </label>
                    </>
                )}

                {/* Usage Hint */}
                <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <h4 className="text-xs font-semibold text-blue-900 dark:text-blue-300 mb-1">
                        Input Data Format
                    </h4>
                    <p className="text-xs text-blue-700 dark:text-blue-400">
                        {safeConfig.operation === 'read'
                            ? 'For read operation, no input data is needed. Output will be a 2D array.'
                            : 'Supported formats: 2D array [[row1], [row2]], array of objects [{title: "A", link: "B"}], or single object. Use Column Mapping to specify field order for objects.'
                        }
                    </p>
                </div>
            </div>
        </div>
    );
};

export default GoogleSheetsNodeConfigComponent;
