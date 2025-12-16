import React from 'react';
import { Sheet } from 'lucide-react';
import type { GoogleSheetsNodeConfig as GoogleSheetsNodeConfigType } from '@/types/nodeConfigs';
import { VariableAutocomplete } from '@/components/builder/VariableAutocomplete';
import { useTranslation } from '@/store/translations';

interface Props {
    config: GoogleSheetsNodeConfigType;
    nodeId?: string;
    onChange: (config: GoogleSheetsNodeConfigType) => void;
}

export const GoogleSheetsNodeConfigComponent: React.FC<Props> = ({ config, onChange }) => {
    const t = useTranslation();

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
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">{t.nodeConfig.googleSheets.title}</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        {t.nodeConfig.googleSheets.description}
                    </p>
                </div>
            </div>

            {/* Fields */}
            <div className="space-y-3">
                {/* Operation */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleSheets.operation}
                    </span>
                    <select
                        value={safeConfig.operation}
                        onChange={(e) => handleOperationChange(e.target.value as "read" | "write" | "append")}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                    >
                        <option value="read">{t.nodeConfig.googleSheets.operationRead}</option>
                        <option value="write">{t.nodeConfig.googleSheets.operationWrite}</option>
                        <option value="append">{t.nodeConfig.googleSheets.operationAppend}</option>
                    </select>
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleSheets.operationHint}
                    </span>
                </label>

                {/* Spreadsheet ID */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleSheets.spreadsheetId} <span className="text-red-500">{t.nodeConfig.required}</span>
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.spreadsheet_id}
                        onChange={handleSpreadsheetIdChange}
                        placeholder={t.nodeConfig.googleSheets.spreadsheetIdPlaceholder}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm font-mono"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleSheets.spreadsheetIdHint}
                    </span>
                </label>

                {/* Sheet Name */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleSheets.sheetName}
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.sheet_name}
                        onChange={handleSheetNameChange}
                        placeholder={t.nodeConfig.googleSheets.sheetNamePlaceholder}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleSheets.sheetNameHint}
                    </span>
                </label>

                {/* Range */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleSheets.range}
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.range}
                        onChange={handleRangeChange}
                        placeholder={t.nodeConfig.googleSheets.rangePlaceholder}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm font-mono"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleSheets.rangeHint}
                    </span>
                </label>

                {/* Credentials */}
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.googleSheets.credentials} <span className="text-red-500">{t.nodeConfig.required}</span>
                    </span>
                    <textarea
                        value={safeConfig.credentials}
                        onChange={(e) => handleCredentialsChange(e.target.value)}
                        placeholder={t.nodeConfig.googleSheets.credentialsPlaceholder}
                        rows={4}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-xs font-mono resize-y"
                    />
                    <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                        {t.nodeConfig.googleSheets.credentialsHint}
                    </span>
                </label>

                {/* Write/Append Options */}
                {showWriteOptions && (
                    <>
                        {/* Value Input Option */}
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                {t.nodeConfig.googleSheets.valueInputOption}
                            </span>
                            <select
                                value={safeConfig.value_input_option}
                                onChange={(e) => handleValueInputOptionChange(e.target.value as "RAW" | "USER_ENTERED")}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                            >
                                <option value="USER_ENTERED">{t.nodeConfig.googleSheets.valueInputOptionUserEntered}</option>
                                <option value="RAW">{t.nodeConfig.googleSheets.valueInputOptionRaw}</option>
                            </select>
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                {t.nodeConfig.googleSheets.valueInputOptionHint}
                            </span>
                        </label>

                        {/* Major Dimension */}
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                {t.nodeConfig.googleSheets.majorDimension}
                            </span>
                            <select
                                value={safeConfig.major_dimension}
                                onChange={(e) => handleMajorDimensionChange(e.target.value as "ROWS" | "COLUMNS")}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                            >
                                <option value="ROWS">{t.nodeConfig.googleSheets.majorDimensionRows}</option>
                                <option value="COLUMNS">{t.nodeConfig.googleSheets.majorDimensionColumns}</option>
                            </select>
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                {t.nodeConfig.googleSheets.majorDimensionHint}
                            </span>
                        </label>

                        {/* Columns Mapping */}
                        <label className="block">
                            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                                {t.nodeConfig.googleSheets.columnMapping}
                            </span>
                            <VariableAutocomplete
                                value={safeConfig.columns || ''}
                                onChange={handleColumnsChange}
                                placeholder={t.nodeConfig.googleSheets.columnMappingPlaceholder}
                                className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 text-sm font-mono"
                            />
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-1 block">
                                {t.nodeConfig.googleSheets.columnMappingHint}
                            </span>
                        </label>
                    </>
                )}

                {/* Usage Hint */}
                <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <h4 className="text-xs font-semibold text-blue-900 dark:text-blue-300 mb-1">
                        {t.nodeConfig.googleSheets.inputDataFormat}
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
