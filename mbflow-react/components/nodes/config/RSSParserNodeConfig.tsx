import React from 'react';
import {Rss, Info} from 'lucide-react';
import type {RSSParserNodeConfig as RSSParserNodeConfigType} from '@/types/nodeConfigs';
import {VariableAutocomplete} from '@/components/builder/VariableAutocomplete';

interface RSSParserNodeConfigProps {
    config: RSSParserNodeConfigType;
    nodeId?: string;
    onChange: (config: RSSParserNodeConfigType) => void;
}

export const RSSParserNodeConfigComponent: React.FC<RSSParserNodeConfigProps> = ({
    config,
    onChange,
}) => {
    // Ensure config has default values to prevent undefined errors
    const safeConfig: RSSParserNodeConfigType = {
        url: config?.url || '',
        maxItems: config?.maxItems ?? 0,
        includeContent: config?.includeContent ?? false,
    };

    const handleUrlChange = (value: string) => {
        onChange({
            ...safeConfig,
            url: value,
        });
    };

    const handleMaxItemsChange = (value: number) => {
        onChange({
            ...safeConfig,
            maxItems: Math.max(0, value),
        });
    };

    const handleIncludeContentChange = (value: boolean) => {
        onChange({
            ...safeConfig,
            includeContent: value,
        });
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div
                className="bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-900/10 dark:to-amber-900/10 border border-orange-200 dark:border-orange-800 rounded-lg p-4 flex items-start gap-3">
                <Rss className="text-orange-600 dark:text-orange-400 flex-shrink-0 mt-0.5" size={18}/>
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">RSS Parser</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        Fetch and parse RSS/Atom feeds, extract structured data
                    </p>
                </div>
            </div>

            {/* RSS Feed URL */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        RSS Feed URL <span className="text-red-500">*</span>
                    </span>
                    <VariableAutocomplete
                        value={safeConfig.url}
                        onChange={handleUrlChange}
                        placeholder="https://example.com/feed.xml"
                        type="input"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 dark:focus:ring-orange-400 text-sm"
                    />
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        URL of the RSS or Atom feed to parse
                    </p>
                </label>
            </div>

            {/* Max Items */}
            <div className="space-y-3">
                <label className="block">
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
                        Max Items <span className="font-normal text-slate-500 dark:text-slate-400">(optional)</span>
                    </span>
                    <input
                        type="number"
                        min="0"
                        value={safeConfig.maxItems || ''}
                        onChange={(e) => handleMaxItemsChange(parseInt(e.target.value, 10) || 0)}
                        placeholder="0 = all items"
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 dark:focus:ring-orange-400 text-sm"
                    />
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        Maximum number of feed items to return (0 = unlimited)
                    </p>
                </label>
            </div>

            {/* Include Content Checkbox */}
            <div className="space-y-3">
                <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 block">
                    Options
                </span>

                <label className="flex items-center gap-3 cursor-pointer group">
                    <input
                        type="checkbox"
                        checked={safeConfig.includeContent}
                        onChange={(e) => handleIncludeContentChange(e.target.checked)}
                        className="w-4 h-4 text-orange-600 bg-white dark:bg-slate-950 border-slate-300 dark:border-slate-700 rounded focus:ring-orange-500 dark:focus:ring-orange-400"
                    />
                    <div>
                        <span className="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white">
                            Include Full Content
                        </span>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                            Include full article content in addition to summary (uses content:encoded for RSS, content for Atom)
                        </p>
                    </div>
                </label>
            </div>

            {/* Info Box */}
            <div
                className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <div className="flex items-start gap-3">
                    <Info className="text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" size={16}/>
                    <div>
                        <h4 className="text-xs font-bold text-blue-900 dark:text-blue-100 mb-2">Supported Formats</h4>
                        <ul className="text-xs text-slate-700 dark:text-slate-300 space-y-1 list-disc pl-4">
                            <li><strong>RSS 2.0</strong> - Most common blog/news feeds</li>
                            <li><strong>Atom 1.0</strong> - Modern syndication format</li>
                        </ul>
                        <h4 className="text-xs font-bold text-blue-900 dark:text-blue-100 mb-2 mt-3">Auto-Detection</h4>
                        <p className="text-xs text-slate-700 dark:text-slate-300">
                            The parser automatically detects and handles both RSS and Atom feeds.
                            No need to specify the format.
                        </p>
                        <h4 className="text-xs font-bold text-blue-900 dark:text-blue-100 mb-2 mt-3">Use Cases</h4>
                        <ul className="text-xs text-slate-700 dark:text-slate-300 space-y-1 list-disc pl-4">
                            <li>Aggregate news from multiple sources</li>
                            <li>Monitor blog updates and new posts</li>
                            <li>Track product releases or announcements</li>
                            <li>Build content curation pipelines</li>
                        </ul>
                    </div>
                </div>
            </div>

            {/* Output Preview */}
            <div
                className="bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-800 rounded-lg p-3">
                <p className="text-xs text-slate-600 dark:text-slate-400 font-medium mb-2">Output Fields</p>
                <div className="grid grid-cols-2 gap-2 text-xs font-mono">
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-orange-600 dark:text-orange-400">title</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-orange-600 dark:text-orange-400">description</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-orange-600 dark:text-orange-400">link</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">items[]</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-blue-600 dark:text-blue-400">item_count</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-purple-600 dark:text-purple-400">feed_type</code>
                    </div>
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-3 font-medium">Each item contains:</p>
                <div className="grid grid-cols-2 gap-2 text-xs font-mono mt-2">
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">title</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">link</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">description</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">pubDate</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">author</code>
                    </div>
                    <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700">
                        <code className="text-green-600 dark:text-green-400">categories[]</code>
                    </div>
                    {safeConfig.includeContent && (
                        <div className="bg-white dark:bg-slate-800 px-2 py-1 rounded border border-slate-200 dark:border-slate-700 col-span-2">
                            <code className="text-orange-600 dark:text-orange-400">content</code>
                            <span className="text-slate-500 dark:text-slate-400 ml-1">(full article)</span>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default RSSParserNodeConfigComponent;
