import React from 'react';
import {Table, Info, Plus, X} from 'lucide-react';
import type {CSVToJSONNodeConfig as CSVToJSONNodeConfigType} from '@/types/nodeConfigs';
import {VariableAutocomplete} from '@/components/builder/VariableAutocomplete';
import {Button} from '../../ui';

interface Props {
    config: CSVToJSONNodeConfigType;
    nodeId?: string;
    onChange: (config: CSVToJSONNodeConfigType) => void;
}

export const CSVToJSONNodeConfigComponent: React.FC<Props> = ({config, onChange}) => {
    // Ensure config has default values to prevent undefined errors
    const safeConfig: CSVToJSONNodeConfigType = {
        delimiter: config?.delimiter || ',',
        has_header: config?.has_header ?? true,
        custom_headers: config?.custom_headers || [],
        trim_spaces: config?.trim_spaces ?? true,
        skip_empty_rows: config?.skip_empty_rows ?? true,
        input_key: config?.input_key || '',
    };

    const handleDelimiterChange = (value: string) => {
        onChange({...safeConfig, delimiter: value});
    };

    const handleHasHeaderChange = (value: boolean) => {
        onChange({...safeConfig, has_header: value});
    };

    const handleCustomHeadersChange = (headers: string[]) => {
        onChange({...safeConfig, custom_headers: headers});
    };

    const handleTrimSpacesChange = (value: boolean) => {
        onChange({...safeConfig, trim_spaces: value});
    };

    const handleSkipEmptyRowsChange = (value: boolean) => {
        onChange({...safeConfig, skip_empty_rows: value});
    };

    const handleInputKeyChange = (value: string) => {
        onChange({...safeConfig, input_key: value});
    };

    const addCustomHeader = () => {
        const newHeaders = [...(safeConfig.custom_headers || []), ''];
        handleCustomHeadersChange(newHeaders);
    };

    const removeCustomHeader = (index: number) => {
        const newHeaders = [...(safeConfig.custom_headers || [])];
        newHeaders.splice(index, 1);
        handleCustomHeadersChange(newHeaders);
    };

    const updateCustomHeader = (index: number, value: string) => {
        const newHeaders = [...(safeConfig.custom_headers || [])];
        newHeaders[index] = value;
        handleCustomHeadersChange(newHeaders);
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div
                className="bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-900/10 dark:to-blue-900/10 border border-cyan-200 dark:border-cyan-800 rounded-lg p-4 flex items-start gap-3">
                <Table className="text-cyan-600 dark:text-cyan-400 flex-shrink-0 mt-0.5" size={18}/>
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">CSV → JSON</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        Convert CSV data to JSON array of objects
                    </p>
                </div>
            </div>

            {/* Info Box */}
            <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                <div className="flex items-start gap-2">
                    <Info className="text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" size={14}/>
                    <div className="text-xs text-slate-700 dark:text-slate-300">
                        <p className="font-medium mb-1">Example:</p>
                        <code className="block bg-white dark:bg-slate-950 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                            Input: name,age,city\nJohn,30,NYC
                        </code>
                        <code className="block bg-white dark:bg-slate-950 px-2 py-1 rounded border border-slate-200 dark:border-slate-700 mt-1">
                            Output: [{`{"name":"John","age":"30","city":"NYC"}`}]
                        </code>
                    </div>
                </div>
            </div>

            {/* Delimiter */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Delimiter
                    </span>
                    <select
                        value={safeConfig.delimiter}
                        onChange={(e) => handleDelimiterChange(e.target.value)}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 text-sm"
                    >
                        <option value=",">Comma (,)</option>
                        <option value=";">Semicolon (;)</option>
                        <option value="\t">Tab (\t)</option>
                        <option value="|">Pipe (|)</option>
                    </select>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        Character used to separate fields in CSV
                    </p>
                </label>
            </div>

            {/* Input Key */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Input Key (Optional)
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.input_key}
                        onChange={handleInputKeyChange}
                        placeholder="e.g., csv, data, content"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 text-sm"
                    />
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        If input is a map, specify the key containing CSV data. Auto-detects common keys (csv, data, content) if empty.
                    </p>
                </label>
            </div>

            {/* Headers Configuration */}
            <div className="space-y-3">
                <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 block">
                    Headers Configuration
                </span>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={safeConfig.has_header}
                        onChange={(e) => handleHasHeaderChange(e.target.checked)}
                        className="w-4 h-4 text-cyan-600 bg-slate-100 dark:bg-slate-900 border-slate-300 dark:border-slate-700 rounded focus:ring-cyan-500 focus:ring-2"
                    />
                    <span className="text-sm text-slate-700 dark:text-slate-300">
                        First row contains headers
                    </span>
                </label>

                {!safeConfig.has_header && (
                    <div className="space-y-2 pl-6">
                        <div className="flex items-center justify-between">
                            <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
                                Custom Headers
                            </span>
                            <Button
                                type="button"
                                onClick={addCustomHeader}
                                variant="outline"
                                size="sm"
                                icon={<Plus size={14}/>}
                                iconPosition="left"
                            >
                                Add Header
                            </Button>
                        </div>

                        {safeConfig.custom_headers && safeConfig.custom_headers.length > 0 ? (
                            <div className="space-y-2">
                                {safeConfig.custom_headers.map((header, index) => (
                                    <div key={index} className="flex items-center gap-2">
                                        <input
                                            type="text"
                                            value={header}
                                            onChange={(e) => updateCustomHeader(index, e.target.value)}
                                            placeholder={`Header ${index + 1}`}
                                            className="flex-1 px-3 py-1.5 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 text-sm"
                                        />
                                        <Button
                                            type="button"
                                            onClick={() => removeCustomHeader(index)}
                                            variant="danger"
                                            size="sm"
                                            icon={<X size={14}/>}
                                        />
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="text-xs text-slate-500 dark:text-slate-400 italic">
                                No custom headers. Auto-generated names (col_0, col_1, ...) will be used.
                            </p>
                        )}
                    </div>
                )}
            </div>

            {/* Options */}
            <div className="space-y-3">
                <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 block">
                    Processing Options
                </span>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={safeConfig.trim_spaces}
                        onChange={(e) => handleTrimSpacesChange(e.target.checked)}
                        className="w-4 h-4 text-cyan-600 bg-slate-100 dark:bg-slate-900 border-slate-300 dark:border-slate-700 rounded focus:ring-cyan-500 focus:ring-2"
                    />
                    <span className="text-sm text-slate-700 dark:text-slate-300">
                        Trim leading/trailing spaces
                    </span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={safeConfig.skip_empty_rows}
                        onChange={(e) => handleSkipEmptyRowsChange(e.target.checked)}
                        className="w-4 h-4 text-cyan-600 bg-slate-100 dark:bg-slate-900 border-slate-300 dark:border-slate-700 rounded focus:ring-cyan-500 focus:ring-2"
                    />
                    <span className="text-sm text-slate-700 dark:text-slate-300">
                        Skip empty rows
                    </span>
                </label>
            </div>
        </div>
    );
};

export default CSVToJSONNodeConfigComponent;
