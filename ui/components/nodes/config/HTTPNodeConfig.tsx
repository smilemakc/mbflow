/**
 * HTTPNodeConfig - React component for configuring HTTP request nodes
 *
 * Ported from: /mbflow-ui/src/components/nodes/config/HTTPNodeConfig.vue
 *
 * Features:
 * - HTTP method selection (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
 * - URL input with template variable support
 * - Key-value header editor (simple implementation)
 * - JSON body editor (textarea for POST/PUT/PATCH methods)
 * - Timeout and retry configuration
 * - Follow redirects toggle
 *
 * Usage:
 * ```tsx
 * <HTTPNodeConfig
 *   config={httpConfig}
 *   nodeId="node-123"
 *   onChange={(newConfig) => console.log(newConfig)}
 * />
 * ```
 */

import React, {useEffect, useState} from "react";
import {HTTP_METHODS, HTTPNodeConfig} from "@/types/nodeConfigs.ts";
import {Button} from '@/components/ui';
import {useTranslation} from '@/store/translations';

interface HTTPNodeConfigProps {
    config: HTTPNodeConfig;
    nodeId?: string;
    onChange: (config: HTTPNodeConfig) => void;
}

export const HTTPNodeConfigComponent: React.FC<HTTPNodeConfigProps> = ({
                                                                           config,
                                                                           nodeId,
                                                                           onChange,
                                                                       }) => {
    const t = useTranslation();
    const [localConfig, setLocalConfig] = useState<HTTPNodeConfig>({
        ...config,
        headers: config.headers ?? {},
        body: config.body ?? "",
    });

    useEffect(() => {
        const newConfig = {
            ...config,
            headers: config.headers ?? {},
            body: config.body ?? "",
        };

        if (JSON.stringify(newConfig) !== JSON.stringify(localConfig)) {
            setLocalConfig(newConfig);
        }
    }, [config]);

    useEffect(() => {
        if (JSON.stringify(localConfig) !== JSON.stringify(config)) {
            onChange(localConfig);
        }
    }, [localConfig]);

    const updateConfig = (updates: Partial<HTTPNodeConfig>) => {
        setLocalConfig((prev) => ({...prev, ...updates}));
    };

    const shouldShowBody = ["POST", "PUT", "PATCH"].includes(localConfig.method);

    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700">
                    {t.nodeConfig.http.method}
                </label>
                <select
                    value={localConfig.method}
                    onChange={(e) =>
                        updateConfig({
                            method: e.target.value as HTTPNodeConfig["method"],
                        })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                >
                    {HTTP_METHODS.map((method) => (
                        <option key={method} value={method}>
                            {method}
                        </option>
                    ))}
                </select>
            </div>

            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700">{t.nodeConfig.http.url}</label>
                <input
                    type="text"
                    value={localConfig.url}
                    onChange={(e) => updateConfig({url: e.target.value})}
                    placeholder={t.nodeConfig.http.urlPlaceholder}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                />
                <span className="text-xs text-gray-500">
          {t.nodeConfig.http.urlHint}
        </span>
            </div>

            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700">{t.nodeConfig.http.headers}</label>
                <div className="space-y-2">
                    {Object.entries(localConfig.headers || {}).map(([key, value], index) => (
                        <div key={index} className="flex gap-2">
                            <input
                                type="text"
                                value={key}
                                onChange={(e) => {
                                    const newHeaders = {...localConfig.headers};
                                    delete newHeaders[key];
                                    newHeaders[e.target.value] = value;
                                    updateConfig({headers: newHeaders});
                                }}
                                placeholder={t.nodeConfig.http.headerKeyPlaceholder}
                                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                            />
                            <input
                                type="text"
                                value={value}
                                onChange={(e) => {
                                    const newHeaders = {...localConfig.headers};
                                    newHeaders[key] = e.target.value;
                                    updateConfig({headers: newHeaders});
                                }}
                                placeholder={t.nodeConfig.http.headerValuePlaceholder}
                                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                            />
                            <Button
                                onClick={() => {
                                    const newHeaders = {...localConfig.headers};
                                    delete newHeaders[key];
                                    updateConfig({headers: newHeaders});
                                }}
                                variant="danger"
                                size="sm"
                                title={t.nodeConfig.http.removeHeader}
                            >
                                ×
                            </Button>
                        </div>
                    ))}
                    <Button
                        onClick={() => {
                            const newHeaders = {...localConfig.headers};
                            let counter = 1;
                            while (newHeaders[`header${counter}`]) {
                                counter++;
                            }
                            newHeaders[`header${counter}`] = "";
                            updateConfig({headers: newHeaders});
                        }}
                        variant="outline"
                        size="sm"
                        fullWidth
                    >
                        {t.nodeConfig.http.addHeader}
                    </Button>
                </div>
            </div>

            {shouldShowBody && (
                <div className="flex flex-col gap-1.5">
                    <label className="text-sm font-semibold text-gray-700">{t.nodeConfig.http.body}</label>
                    <textarea
                        value={localConfig.body || ""}
                        onChange={(e) => updateConfig({body: e.target.value})}
                        placeholder={t.nodeConfig.http.bodyPlaceholder}
                        rows={6}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm font-mono transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100 resize-y"
                    />
                    <span className="text-xs text-gray-500">
            {t.nodeConfig.http.bodyHint}
          </span>
                </div>
            )}

            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700">
                    {t.nodeConfig.http.timeout}
                </label>
                <input
                    type="number"
                    value={localConfig.timeout_seconds ?? 30}
                    onChange={(e) =>
                        updateConfig({timeout_seconds: Number(e.target.value)})
                    }
                    min={1}
                    max={300}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                />
            </div>

            <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-gray-700">
                    {t.nodeConfig.http.retryCount}
                </label>
                <input
                    type="number"
                    value={localConfig.retry_count ?? 0}
                    onChange={(e) =>
                        updateConfig({retry_count: Number(e.target.value)})
                    }
                    min={0}
                    max={10}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm transition-colors focus:outline-none focus:border-blue-500 focus:ring-3 focus:ring-blue-100"
                />
            </div>

            <div className="flex flex-col gap-1.5">
                <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={localConfig.follow_redirects ?? true}
                        onChange={(e) =>
                            updateConfig({follow_redirects: e.target.checked})
                        }
                        className="w-[18px] h-[18px] cursor-pointer"
                    />
                    {t.nodeConfig.http.followRedirects}
                </label>
            </div>
        </div>
    );
};

export default HTTPNodeConfigComponent;
