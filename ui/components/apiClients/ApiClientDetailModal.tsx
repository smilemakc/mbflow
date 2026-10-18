import React, {useEffect, useMemo, useRef, useState} from 'react';
import {AlertTriangle, ChevronRight, Download, Loader2, Search, Trash2, Unlock, Upload} from 'lucide-react';
import {Button, ConfirmModal, Modal} from '@/components/ui';
import {FormField} from '@/components/ui/form';
import {configStyles} from '@/styles/configStyles';
import {useTranslation} from '@/store/translations';
import {useApiClientStore} from '@/store/apiClientStore';
import {getErrorMessage} from '@/lib/api';
import {toast} from '@/lib/toast';
import {apiClientService, ApiClient, ApiClientAuth, ApiOperation} from '@/services/apiClientService';
import {AuthSettingsForm} from './AuthSettingsForm';
import {BaseUrlField} from './BaseUrlField';
import {KeyValueRows} from './KeyValueRows';
import {MethodBadge} from './MethodBadge';
import {describeSchema, format, groupByTag, operationMatches, successResponse} from './schemaUtils';

interface ApiClientDetailModalProps {
    clientId: string | null;
    onClose: () => void;
}

type Tab = 'operations' | 'settings';

export const ApiClientDetailModal: React.FC<ApiClientDetailModalProps> = ({clientId, onClose}) => {
    const t = useTranslation();
    const {details, loadClient, upsertClient, removeClient} = useApiClientStore();
    const client = clientId ? details[clientId] : undefined;
    const [tab, setTab] = useState<Tab>('operations');
    const [confirmDelete, setConfirmDelete] = useState(false);
    const [isReplacing, setIsReplacing] = useState(false);
    const [removedOperations, setRemovedOperations] = useState<string[]>([]);
    const specInputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (clientId) {
            setTab('operations');
            setRemovedOperations([]);
            void loadClient(clientId, true);
        }
    }, [clientId, loadClient]);

    const replaceSpec = async (file: File) => {
        if (!client) return;
        setIsReplacing(true);
        try {
            const response = await apiClientService.replaceSpec(client.id, file);
            upsertClient(response.data.client);
            const {added, removed} = response.data.changes;
            setRemovedOperations(removed);
            toast.success(t.apiClients.specUpdated, format(t.apiClients.specChanges, {added: added.length, removed: removed.length}));
        } catch (error) {
            toast.error(t.common.error, getErrorMessage(error));
        } finally {
            setIsReplacing(false);
        }
    };

    const downloadSpec = async () => {
        if (!client) return;
        try {
            const response = await apiClientService.downloadSpec(client.id);
            const disposition = String(response.headers['content-disposition'] ?? '');
            const filename = /filename="?([^";]+)"?/.exec(disposition)?.[1] ?? `${client.name}.yaml`;
            const url = URL.createObjectURL(response.data);
            const link = document.createElement('a');
            link.href = url;
            link.download = filename;
            link.click();
            URL.revokeObjectURL(url);
        } catch (error) {
            toast.error(t.common.error, getErrorMessage(error));
        }
    };

    const deleteClient = async () => {
        if (!client) return;
        try {
            await apiClientService.remove(client.id);
            removeClient(client.id);
            toast.success(t.apiClients.deleted);
            setConfirmDelete(false);
            onClose();
        } catch (error) {
            toast.error(t.common.error, getErrorMessage(error));
        }
    };

    const title = client ? (
        <div className="flex flex-wrap items-baseline gap-x-2">
            <span>{client.name}</span>
            <span className="text-xs font-normal text-slate-500">
                {client.title}{client.api_version ? ` v${client.api_version}` : ''}
                {' · '}
                {client.spec_version === '2.0' ? 'Swagger 2.0' : `OpenAPI ${client.spec_version}`}
            </span>
        </div>
    ) : t.common.loading;

    const footer = client && (
        <div className="flex flex-wrap items-center justify-between gap-2">
            <Button variant="danger" size="sm" icon={<Trash2 size={14}/>} onClick={() => setConfirmDelete(true)}>
                {t.apiClients.delete}
            </Button>
            <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" icon={<Download size={14}/>} onClick={downloadSpec}>
                    {t.apiClients.downloadSpec}
                </Button>
                <Button
                    variant="outline"
                    size="sm"
                    icon={<Upload size={14}/>}
                    loading={isReplacing}
                    onClick={() => specInputRef.current?.click()}
                >
                    {t.apiClients.replaceSpec}
                </Button>
            </div>
        </div>
    );

    return (
        <>
            <Modal isOpen={!!clientId} onClose={onClose} title={title} size="full" footer={footer}>
                <input
                    ref={specInputRef}
                    type="file"
                    accept=".json,.yaml,.yml"
                    className="hidden"
                    onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) void replaceSpec(file);
                        e.target.value = '';
                    }}
                />
                {!client ? (
                    <div className="flex justify-center py-12">
                        <Loader2 size={28} className="animate-spin text-blue-500"/>
                    </div>
                ) : (
                    <div className="space-y-4">
                        {removedOperations.length > 0 && (
                            <div className={`${configStyles.infoBox.warning} text-xs text-amber-800 dark:text-amber-300`}>
                                <p className="font-semibold mb-1">{t.apiClients.removedOperationsWarning}</p>
                                <p className="font-mono break-words">{removedOperations.join(', ')}</p>
                            </div>
                        )}
                        {client.warnings.length > 0 && (
                            <details className={`${configStyles.infoBox.warning} text-xs text-amber-800 dark:text-amber-300`}>
                                <summary className="cursor-pointer font-semibold">
                                    {t.apiClients.warnings} ({client.warnings.length})
                                </summary>
                                <ul className="list-disc pl-4 mt-1 space-y-0.5">
                                    {client.warnings.map((w, i) => <li key={i} className="break-words">{w.message}</li>)}
                                </ul>
                            </details>
                        )}

                        <div className="flex gap-1 border-b border-slate-200 dark:border-slate-800" role="tablist">
                            {(['operations', 'settings'] as Tab[]).map((value) => (
                                <button
                                    key={value}
                                    type="button"
                                    role="tab"
                                    aria-selected={tab === value}
                                    onClick={() => setTab(value)}
                                    className={`px-3 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
                                        tab === value
                                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                            : 'border-transparent text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'
                                    }`}
                                >
                                    {value === 'operations'
                                        ? `${t.apiClients.operationsTab} (${client.operations.length})`
                                        : t.apiClients.settingsTab}
                                </button>
                            ))}
                        </div>

                        {tab === 'operations'
                            ? <OperationsList client={client}/>
                            : <ClientSettings client={client} onSaved={upsertClient}/>}
                    </div>
                )}
            </Modal>

            <ConfirmModal
                isOpen={confirmDelete}
                onClose={() => setConfirmDelete(false)}
                onConfirm={deleteClient}
                title={t.apiClients.deleteTitle}
                message={t.apiClients.deleteMessage}
                confirmText={t.common.delete}
                variant="danger"
            />
        </>
    );
};

const OperationsList: React.FC<{client: ApiClient}> = ({client}) => {
    const t = useTranslation();
    const [query, setQuery] = useState('');
    const [expanded, setExpanded] = useState<string | null>(null);

    const groups = useMemo(
        () => groupByTag(client.operations.filter((op) => operationMatches(op, query)), t.apiClients.untagged),
        [client.operations, query, t.apiClients.untagged],
    );

    return (
        <div className="space-y-3">
            <div className="relative">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"/>
                <input
                    type="text"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    placeholder={t.apiClients.searchOperations}
                    className={`${configStyles.input} pl-8`}
                />
            </div>
            {groups.length === 0 && (
                <p className="py-6 text-center text-sm text-slate-500">{t.apiClients.noOperationsFound}</p>
            )}
            <div className="max-h-[55vh] overflow-y-auto space-y-4 pr-1">
                {groups.map(([tag, operations]) => (
                    <section key={tag}>
                        <h4 className="mb-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-500">{tag}</h4>
                        <div className="divide-y divide-slate-100 dark:divide-slate-800 rounded-lg border border-slate-200 dark:border-slate-800">
                            {operations.map((op) => (
                                <OperationRow
                                    key={op.id}
                                    operation={op}
                                    client={client}
                                    expanded={expanded === op.id}
                                    onToggle={() => setExpanded(expanded === op.id ? null : op.id)}
                                />
                            ))}
                        </div>
                    </section>
                ))}
            </div>
        </div>
    );
};

const OperationRow: React.FC<{
    operation: ApiOperation;
    client: ApiClient;
    expanded: boolean;
    onToggle: () => void;
}> = ({operation, client, expanded, onToggle}) => {
    const t = useTranslation();
    const response = successResponse(operation);
    const body = operation.request_body?.content?.[0];

    return (
        <div>
            <button
                type="button"
                onClick={onToggle}
                aria-expanded={expanded}
                className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-slate-50 dark:hover:bg-slate-800/50"
            >
                <ChevronRight size={14} className={`shrink-0 text-slate-400 transition-transform ${expanded ? 'rotate-90' : ''}`}/>
                <MethodBadge method={operation.method}/>
                <span className={`font-mono text-xs truncate ${operation.deprecated ? 'line-through text-slate-400' : 'text-slate-800 dark:text-slate-200'}`}>
                    {operation.path}
                </span>
                <span className="ml-auto pl-2 text-xs text-slate-500 truncate max-w-[45%]">{operation.summary}</span>
                {operation.no_auth && client.auth.type !== 'none' && (
                    <span title={t.apiClients.noAuthBadge} className="shrink-0 text-slate-400"><Unlock size={12}/></span>
                )}
            </button>
            {expanded && (
                <div className="px-9 pb-3 space-y-2 text-xs text-slate-600 dark:text-slate-300">
                    <p className="font-mono text-[11px] text-slate-400">{operation.id}</p>
                    {operation.description && <p className="whitespace-pre-line">{operation.description}</p>}
                    {operation.deprecated && (
                        <p className="flex items-center gap-1 text-amber-600"><AlertTriangle size={12}/>{t.apiClients.deprecated}</p>
                    )}
                    {(operation.parameters ?? []).length > 0 && (
                        <ul className="space-y-0.5">
                            {operation.parameters!.map((p) => (
                                <li key={`${p.in}:${p.name}`} className="font-mono">
                                    <span className="text-slate-400">{p.in}</span>{' '}
                                    {p.name}{p.required && <span className="text-red-500">*</span>}
                                    <span className="text-slate-400">: {describeSchema(p.schema, client.schemas)}</span>
                                </li>
                            ))}
                        </ul>
                    )}
                    {body && (
                        <p className="font-mono">
                            <span className="text-slate-400">body</span> {body.content_type}
                            {body.schema && <span className="text-slate-400">: {describeSchema(body.schema, client.schemas)}</span>}
                        </p>
                    )}
                    {response && (
                        <p className="font-mono">
                            <span className="text-slate-400">→ {response.status}</span>{' '}
                            {response.content?.[0]?.schema
                                ? describeSchema(response.content[0].schema, client.schemas)
                                : response.description}
                        </p>
                    )}
                </div>
            )}
        </div>
    );
};

const ClientSettings: React.FC<{client: ApiClient; onSaved: (client: ApiClient) => void}> = ({client, onSaved}) => {
    const t = useTranslation();
    const [name, setName] = useState(client.name);
    const [description, setDescription] = useState(client.description ?? '');
    const [baseUrl, setBaseUrl] = useState(client.base_url ?? '');
    const [auth, setAuth] = useState<ApiClientAuth>(client.auth?.type ? client.auth : {type: 'none'});
    const [headers, setHeaders] = useState<Record<string, string>>(client.default_headers ?? {});
    const [isSaving, setIsSaving] = useState(false);

    useEffect(() => {
        setName(client.name);
        setDescription(client.description ?? '');
        setBaseUrl(client.base_url ?? '');
        setAuth(client.auth?.type ? client.auth : {type: 'none'});
        setHeaders(client.default_headers ?? {});
    }, [client.id, client.updated_at]);

    const save = async () => {
        setIsSaving(true);
        try {
            const response = await apiClientService.update(client.id, {
                name,
                description,
                base_url: baseUrl,
                auth,
                default_headers: headers,
            });
            onSaved(response.data);
            toast.success(t.apiClients.saved);
        } catch (error) {
            toast.error(t.common.error, getErrorMessage(error));
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-4">
                <FormField label={t.apiClients.name} required>
                    <input
                        type="text"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        className={configStyles.input}
                        maxLength={255}
                    />
                </FormField>
                <FormField label={t.apiClients.description}>
                    <textarea
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        rows={3}
                        className={configStyles.textarea}
                    />
                </FormField>
                <BaseUrlField value={baseUrl} onChange={setBaseUrl} servers={client.servers}/>
                <FormField label={t.apiClients.defaultHeaders} hint={t.apiClients.defaultHeadersHint}>
                    <KeyValueRows
                        value={headers}
                        onChange={setHeaders}
                        addLabel={t.apiClients.addHeader}
                        keyPlaceholder={t.apiClients.headerName}
                        valuePlaceholder={t.apiClients.headerValue}
                        resetKey={`${client.id}:${client.updated_at}`}
                    />
                </FormField>
            </div>
            <div className="space-y-4">
                <div className={configStyles.section}>
                    <h4 className={configStyles.sectionTitle}>{t.apiClients.auth}</h4>
                    <AuthSettingsForm value={auth} onChange={setAuth} securitySchemes={client.security_schemes}/>
                </div>
                <div className="flex justify-end">
                    <Button onClick={save} loading={isSaving} disabled={!name.trim()}>
                        {t.apiClients.save}
                    </Button>
                </div>
            </div>
        </div>
    );
};
