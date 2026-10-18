import React from 'react';
import {AlertTriangle} from 'lucide-react';
import {FormField} from '@/components/ui/form';
import {configStyles} from '@/styles/configStyles';
import {useTranslation} from '@/store/translations';
import type {ApiServer} from '@/services/apiClientService';

const isAbsoluteUrl = (url: string) => /^https?:\/\/[^/]+/i.test(url);

interface BaseUrlFieldProps {
    value: string;
    onChange: (value: string) => void;
    servers: ApiServer[];
}

/** Base URL input with the absolute servers of the specification as one-click choices. */
export const BaseUrlField: React.FC<BaseUrlFieldProps> = ({value, onChange, servers}) => {
    const t = useTranslation();
    const absolute = servers.filter((s) => isAbsoluteUrl(s.url));

    return (
        <FormField label={t.apiClients.baseUrl} hint={t.apiClients.baseUrlHint}>
            <input
                type="url"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                className={`${configStyles.input} font-mono`}
                placeholder="https://api.example.com/v1"
            />
            {absolute.length > 0 && (
                <div className="flex flex-wrap gap-1.5 mt-2">
                    {absolute.map((server) => (
                        <button
                            key={server.url}
                            type="button"
                            onClick={() => onChange(server.url)}
                            title={server.description}
                            className={`max-w-full truncate px-2 py-0.5 rounded-md border text-[11px] font-mono transition-colors ${
                                value === server.url
                                    ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 hover:border-blue-400'
                            }`}
                        >
                            {server.url}
                        </button>
                    ))}
                </div>
            )}
            {!value.trim() && (
                <p className="flex items-center gap-1 mt-2 text-xs text-amber-600 dark:text-amber-400">
                    <AlertTriangle size={12}/>
                    {t.apiClients.baseUrlMissing}
                </p>
            )}
        </FormField>
    );
};
