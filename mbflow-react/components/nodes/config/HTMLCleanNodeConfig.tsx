import React from 'react';
import {FileText, Info} from 'lucide-react';
import type {HTMLCleanNodeConfig as HTMLCleanNodeConfigType} from '@/types/nodeConfigs';
import {VariableAutocomplete} from '@/components/builder/VariableAutocomplete';
import { useTranslation } from '@/store/translations';

interface HTMLCleanNodeConfigProps {
    config: HTMLCleanNodeConfigType;
    nodeId?: string;
    onChange: (config: HTMLCleanNodeConfigType) => void;
}

export const HTMLCleanNodeConfigComponent: React.FC<HTMLCleanNodeConfigProps> = ({
                                                                                     config,
                                                                                     onChange,
                                                                                 }) => {
    const t = useTranslation();

    // Ensure config has default values to prevent undefined errors
    const safeConfig: HTMLCleanNodeConfigType = {
        output_format: config?.output_format || 'both',
        extract_metadata: config?.extract_metadata ?? true,
        preserve_links: config?.preserve_links ?? false,
        max_length: config?.max_length || 0,
        input_key: config?.input_key || '',
    };

    const handleInputKeyChange = (value: string) => {
        onChange({
            ...safeConfig,
            input_key: value,
        });
    };

    const handleOutputFormatChange = (format: 'text' | 'html' | 'both') => {
        onChange({
            ...safeConfig,
            output_format: format,
        });
    };

    const handleExtractMetadataChange = (value: boolean) => {
        onChange({
            ...safeConfig,
            extract_metadata: value,
        });
    };

    const handlePreserveLinksChange = (value: boolean) => {
        onChange({
            ...safeConfig,
            preserve_links: value,
        });
    };

    const handleMaxLengthChange = (value: number) => {
        onChange({
            ...safeConfig,
            max_length: Math.max(0, value),
        });
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div
                className="bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-900/10 dark:to-amber-900/10 border border-orange-200 dark:border-orange-800 rounded-lg p-4 flex items-start gap-3">
                <FileText className="text-orange-600 dark:text-orange-400 flex-shrink-0 mt-0.5" size={18}/>
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">{t.nodeConfig.htmlClean.title}</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        {t.nodeConfig.htmlClean.description}
                    </p>
                </div>
            </div>

            {/* Output Format */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.htmlClean.outputFormat}
                    </span>
                    <select
                        value={safeConfig.output_format}
                        onChange={(e) => handleOutputFormatChange(e.target.value as 'text' | 'html' | 'both')}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 dark:focus:ring-orange-400 text-sm"
                    >
                        <option value="both">{t.nodeConfig.htmlClean.outputFormatBoth}</option>
                        <option value="text">{t.nodeConfig.htmlClean.outputFormatText}</option>
                        <option value="html">{t.nodeConfig.htmlClean.outputFormatHtml}</option>
                    </select>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        {t.nodeConfig.htmlClean.outputFormatHint}
                    </p>
                </label>
            </div>

            {/* Checkboxes */}
            <div className="space-y-3">
                <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 block">
                    {t.nodeConfig.htmlClean.options}
                </span>

                {/* Extract Metadata */}
                <label className="flex items-center gap-3 cursor-pointer group">
                    <input
                        type="checkbox"
                        checked={safeConfig.extract_metadata}
                        onChange={(e) => handleExtractMetadataChange(e.target.checked)}
                        className="w-4 h-4 text-orange-600 bg-white dark:bg-slate-950 border-slate-300 dark:border-slate-700 rounded focus:ring-orange-500 dark:focus:ring-orange-400"
                    />
                    <div>
                        <span className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white">
                            {t.nodeConfig.htmlClean.extractMetadata}
                        </span>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                            {t.nodeConfig.htmlClean.extractMetadataHint}
                        </p>
                    </div>
                </label>

                {/* Preserve Links */}
                <label className="flex items-center gap-3 cursor-pointer group">
                    <input
                        type="checkbox"
                        checked={safeConfig.preserve_links}
                        onChange={(e) => handlePreserveLinksChange(e.target.checked)}
                        className="w-4 h-4 text-orange-600 bg-white dark:bg-slate-950 border-slate-300 dark:border-slate-700 rounded focus:ring-orange-500 dark:focus:ring-orange-400"
                    />
                    <div>
                        <span className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white">
                            {t.nodeConfig.htmlClean.preserveLinks}
                        </span>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                            {t.nodeConfig.htmlClean.preserveLinksHint}
                        </p>
                    </div>
                </label>
            </div>

            {/* Max Length */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.htmlClean.maxLength} <span className="font-normal text-slate-500 dark:text-slate-400">{t.nodeConfig.optional}</span>
                    </span>
                    <input
                        type="number"
                        min="0"
                        value={safeConfig.max_length || ''}
                        onChange={(e) => handleMaxLengthChange(parseInt(e.target.value, 10) || 0)}
                        placeholder={t.nodeConfig.htmlClean.maxLengthPlaceholder}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 dark:focus:ring-orange-400 text-sm"
                    />
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        {t.nodeConfig.htmlClean.maxLengthHint}
                    </p>
                </label>
            </div>

            {/* Input Key */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        {t.nodeConfig.htmlClean.inputKey} <span className="font-normal text-slate-500 dark:text-slate-400">{t.nodeConfig.optional}</span>
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.input_key}
                        onChange={handleInputKeyChange}
                        placeholder={t.nodeConfig.htmlClean.inputKeyPlaceholder}
                        type="input"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 dark:focus:ring-orange-400 text-sm"
                    />
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        {t.nodeConfig.htmlClean.inputKeyHint}
                    </p>
                </label>
            </div>

            {/* Info Box */}
            <div
                className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <div className="flex items-start gap-3">
                    <Info className="text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" size={16}/>
                    <div>
                        <h4 className="text-xs font-bold text-blue-900 dark:text-blue-100 mb-2">{t.nodeConfig.htmlClean.smartDetection}</h4>
                        <p className="text-xs text-slate-700 dark:text-slate-300 mb-3">
                            Automatically detects if input is HTML. Non-HTML content (plain text, JSON, etc.) passes through unchanged.
                        </p>
                        <h4 className="text-xs font-bold text-blue-900 dark:text-blue-100 mb-2">{t.nodeConfig.htmlClean.whatGetsRemoved}</h4>
                        <ul className="text-xs text-slate-700 dark:text-slate-300 space-y-1 list-disc pl-4">
                            <li>Scripts, styles, and inline JavaScript</li>
                            <li>Navigation, sidebars, and footers</li>
                            <li>Ads, social buttons, and related content</li>
                            <li>Comments and cookie notices</li>
                        </ul>
                        <h4 className="text-xs font-bold text-blue-900 dark:text-blue-100 mb-2 mt-3">{t.nodeConfig.htmlClean.whatGetsKept}</h4>
                        <ul className="text-xs text-slate-700 dark:text-slate-300 space-y-1 list-disc pl-4">
                            <li>Main article content and headings</li>
                            <li>Lists, tables, and code blocks</li>
                            <li>Links and images (with alt text)</li>
                        </ul>
                    </div>
                </div>
            </div>

            {/* Output Preview */}
            <div
                className="bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-800 rounded-lg p-3">
                <p className="text-xs text-slate-600 dark:text-slate-400 font-medium mb-2">{t.nodeConfig.htmlClean.outputFields}</p>
                <div className="grid grid-cols-2 gap-2 text-xs font-mono">
                    {(safeConfig.output_format === 'both' || safeConfig.output_format === 'text') && (
                        <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                            <code className="text-orange-600 dark:text-orange-400">text_content</code>
                        </div>
                    )}
                    {(safeConfig.output_format === 'both' || safeConfig.output_format === 'html') && (
                        <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                            <code className="text-orange-600 dark:text-orange-400">html_content</code>
                        </div>
                    )}
                    {safeConfig.extract_metadata && (
                        <>
                            <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                                <code className="text-blue-600 dark:text-blue-400">title</code>
                            </div>
                            <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                                <code className="text-blue-600 dark:text-blue-400">author</code>
                            </div>
                            <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                                <code className="text-blue-600 dark:text-blue-400">excerpt</code>
                            </div>
                            <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                                <code className="text-blue-600 dark:text-blue-400">site_name</code>
                            </div>
                        </>
                    )}
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">length</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">word_count</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-purple-600 dark:text-purple-400">is_html</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-purple-600 dark:text-purple-400">passthrough</code>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default HTMLCleanNodeConfigComponent;
