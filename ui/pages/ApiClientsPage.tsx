import React, {useEffect, useState} from 'react';
import {AlertTriangle, KeyRound, Loader2, Plug, Plus, RefreshCw} from 'lucide-react';
import {Button} from '@/components/ui';
import {ApiClientDetailModal, ImportApiClientModal} from '@/components/apiClients';
import {format} from '@/components/apiClients/schemaUtils';
import {useApiClientStore} from '@/store/apiClientStore';
import {useTranslation} from '@/store/translations';
import type {ApiClientSummary} from '@/services/apiClientService';

export const ApiClientsPage: React.FC = () => {
    const t = useTranslation();
    const {clients, isLoading, isLoaded, error, loadClients} = useApiClientStore();
    const [showImport, setShowImport] = useState(false);
    const [openClientId, setOpenClientId] = useState<string | null>(null);

    useEffect(() => {
        void loadClients(true);
    }, [loadClients]);

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-7xl mx-auto space-y-6">
                <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t.apiClients.title}</h1>
                        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400 max-w-2xl">{t.apiClients.subtitle}</p>
                    </div>
                    <div className="flex gap-2">
                        <Button variant="outline" icon={<RefreshCw size={16}/>} onClick={() => loadClients(true)} disabled={isLoading}>
                            {t.common.refresh}
                        </Button>
                        <Button icon={<Plus size={16}/>} onClick={() => setShowImport(true)}>
                            {t.apiClients.import}
                        </Button>
                    </div>
                </div>

                {error && (
                    <div className="flex items-center gap-2 p-4 rounded-lg border border-red-200 bg-red-50 text-sm text-red-700 dark:border-red-900 dark:bg-red-900/20 dark:text-red-300">
                        <AlertTriangle size={16}/>
                        {t.apiClients.loadFailed}: {error}
                    </div>
                )}

                {isLoading && !isLoaded && (
                    <div className="flex justify-center py-16">
                        <Loader2 size={32} className="animate-spin text-blue-500"/>
                    </div>
                )}

                {isLoaded && clients.length === 0 && (
                    <div className="flex flex-col items-center text-center px-6 py-16 rounded-2xl border-2 border-dashed border-slate-200 dark:border-slate-800">
                        <div className="p-4 rounded-2xl bg-indigo-50 dark:bg-indigo-900/30 mb-4">
                            <Plug size={32} className="text-indigo-500"/>
                        </div>
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">{t.apiClients.emptyTitle}</h2>
                        <p className="mt-2 max-w-md text-sm text-slate-500 dark:text-slate-400">{t.apiClients.emptyDescription}</p>
                        <Button className="mt-6" icon={<Plus size={16}/>} onClick={() => setShowImport(true)}>
                            {t.apiClients.import}
                        </Button>
                    </div>
                )}

                {clients.length > 0 && (
                    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                        {clients.map((client) => (
                            <ClientCard key={client.id} client={client} onOpen={() => setOpenClientId(client.id)}/>
                        ))}
                    </div>
                )}
            </div>

            <ImportApiClientModal
                isOpen={showImport}
                onClose={() => setShowImport(false)}
                onImported={(client) => setOpenClientId(client.id)}
            />
            <ApiClientDetailModal clientId={openClientId} onClose={() => setOpenClientId(null)}/>
        </div>
    );
};

const ClientCard: React.FC<{client: ApiClientSummary; onOpen: () => void}> = ({client, onOpen}) => {
    const t = useTranslation();
    const authType = client.auth?.type && client.auth.type !== 'none' ? client.auth.type : null;

    return (
        <button
            type="button"
            onClick={onOpen}
            className="flex flex-col gap-3 p-5 text-left rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 hover:border-blue-400 dark:hover:border-blue-600 hover:shadow-md transition-all"
        >
            <div className="flex items-start gap-3 w-full">
                <div className="p-2 rounded-lg bg-indigo-50 dark:bg-indigo-900/30">
                    <Plug size={18} className="text-indigo-500"/>
                </div>
                <div className="min-w-0 flex-1">
                    <h3 className="font-semibold text-slate-900 dark:text-white truncate">{client.name}</h3>
                    <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                        {client.title}{client.api_version ? ` v${client.api_version}` : ''}
                    </p>
                </div>
                <span className="shrink-0 px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-[11px] font-medium text-slate-600 dark:text-slate-300">
                    {client.spec_version === '2.0' ? 'Swagger 2.0' : `OpenAPI ${client.spec_version}`}
                </span>
            </div>
            {client.description && (
                <p className="text-sm text-slate-600 dark:text-slate-400 line-clamp-2">{client.description}</p>
            )}
            <p className={`text-xs font-mono truncate w-full ${client.base_url ? 'text-slate-500' : 'text-amber-600 dark:text-amber-400'}`}>
                {client.base_url || t.apiClients.baseUrlMissing}
            </p>
            <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                <span className="font-medium text-slate-700 dark:text-slate-200">
                    {format(t.apiClients.operationsCount, {count: client.operation_count})}
                </span>
                {authType && (
                    <span className="inline-flex items-center gap-1">
                        <KeyRound size={12}/>
                        {t.apiClients.authTypes[authType]}
                    </span>
                )}
                {client.warnings.length > 0 && (
                    <span className="inline-flex items-center gap-1 text-amber-600 dark:text-amber-400">
                        <AlertTriangle size={12}/>
                        {client.warnings.length}
                    </span>
                )}
                <span className="ml-auto">
                    {t.apiClients.updatedAt}: {new Date(client.updated_at).toLocaleDateString()}
                </span>
            </div>
        </button>
    );
};
