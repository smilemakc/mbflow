import React, {useEffect, useMemo, useState} from 'react';
import {Link} from 'react-router-dom';
import {AlertTriangle, ChevronRight, Loader2, Plug, Plus, Search, X} from 'lucide-react';
import {useTranslation} from '@/store/translations';
import {useApiClientStore} from '@/store/apiClientStore';
import {configStyles} from '@/styles/configStyles';
import {MethodBadge} from '@/components/apiClients/MethodBadge';
import {format, groupByTag, operationLabel, operationMatches} from '@/components/apiClients/schemaUtils';
import type {ApiToolRef} from '@/types/nodeConfigs';

/** Tools one LLM request supports at most; the backend rejects more. */
const MAX_TOOLS = 128;

interface LLMApiToolsProps {
    value: ApiToolRef[] | undefined;
    /** Functions of the node configured in JSON; they count towards the tool limit. */
    otherTools: number;
    onChange: (value: ApiToolRef[] | undefined) => void;
}

/** Selects operations of imported API clients that the LLM node calls as tools. */
export const LLMApiTools: React.FC<LLMApiToolsProps> = ({value, otherTools, onChange}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.llm.apiTools;
    const refs = useMemo(() => value ?? [], [value]);
    const clients = useApiClientStore((s) => s.clients);
    const isLoaded = useApiClientStore((s) => s.isLoaded);
    const loadClients = useApiClientStore((s) => s.loadClients);
    // Clients added before any of their operations is selected: the config keeps
    // only clients with operations, so the list is kept here.
    const [pending, setPending] = useState<string[]>([]);

    useEffect(() => {
        void loadClients();
    }, [loadClients]);

    const shownIds = useMemo(() => {
        const ids = refs.map((ref) => ref.client_id);
        pending.forEach((id) => {
            if (!ids.includes(id)) ids.push(id);
        });
        return ids;
    }, [refs, pending]);

    const selectedCount = refs.reduce((sum, ref) => sum + ref.operation_ids.length, 0);
    const available = clients.filter((client) => !shownIds.includes(client.id));

    const emit = (next: ApiToolRef[]) => onChange(next.length > 0 ? next : undefined);

    const setOperations = (clientId: string, operationIds: string[]) => {
        const index = refs.findIndex((ref) => ref.client_id === clientId);
        const next = refs.filter((ref) => ref.client_id !== clientId);
        if (operationIds.length > 0) {
            next.splice(index >= 0 ? index : next.length, 0, {client_id: clientId, operation_ids: operationIds});
            setPending((ids) => ids.filter((id) => id !== clientId));
        } else {
            setPending((ids) => (ids.includes(clientId) ? ids : [...ids, clientId]));
        }
        emit(next);
    };

    const removeClient = (clientId: string) => {
        setPending((ids) => ids.filter((id) => id !== clientId));
        emit(refs.filter((ref) => ref.client_id !== clientId));
    };

    return (
        <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                    {tt.title} ({selectedCount})
                </span>
                <Link to="/api-clients" className="text-[11px] text-blue-600 dark:text-blue-400 hover:underline shrink-0">
                    {tt.manage}
                </Link>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400">{tt.hint}</p>

            {shownIds.map((clientId) => (
                <ClientOperations
                    key={clientId}
                    clientId={clientId}
                    selected={refs.find((ref) => ref.client_id === clientId)?.operation_ids ?? []}
                    onChange={(ids) => setOperations(clientId, ids)}
                    onRemove={() => removeClient(clientId)}
                />
            ))}

            {!isLoaded && (
                <div className="flex justify-center py-2"><Loader2 size={16} className="animate-spin text-slate-400"/></div>
            )}
            {isLoaded && clients.length === 0 && shownIds.length === 0 && (
                <div className="p-3 rounded-lg border border-dashed border-slate-300 dark:border-slate-700 text-xs text-slate-500 dark:text-slate-400 space-y-2">
                    <p>{tt.noClients}</p>
                    <Link to="/api-clients" className="inline-flex items-center gap-1 font-medium text-blue-600 dark:text-blue-400 hover:underline">
                        <Plus size={12}/>
                        {tt.importApi}
                    </Link>
                </div>
            )}
            {available.length > 0 && (
                <select
                    value=""
                    onChange={(e) => {
                        const id = e.target.value;
                        if (id) setPending((ids) => (ids.includes(id) ? ids : [...ids, id]));
                    }}
                    className={`${configStyles.select} text-xs`}
                    aria-label={tt.addClient}
                >
                    <option value="">{tt.addClient}</option>
                    {available.map((client) => (
                        <option key={client.id} value={client.id}>{client.name}</option>
                    ))}
                </select>
            )}

            {selectedCount + otherTools > MAX_TOOLS && (
                <p className="flex gap-1.5 text-xs text-amber-700 dark:text-amber-300">
                    <AlertTriangle size={13} className="shrink-0 mt-0.5"/>
                    {format(tt.tooMany, {count: selectedCount + otherTools, max: MAX_TOOLS})}
                </p>
            )}
        </div>
    );
};

const ClientOperations: React.FC<{
    clientId: string;
    selected: string[];
    onChange: (operationIds: string[]) => void;
    onRemove: () => void;
}> = ({clientId, selected, onChange, onRemove}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.llm.apiTools;
    const client = useApiClientStore((s) => s.details[clientId]);
    const missing = useApiClientStore((s) => s.missing[clientId]);
    const loadClient = useApiClientStore((s) => s.loadClient);
    const [query, setQuery] = useState('');
    const [expanded, setExpanded] = useState(selected.length === 0);

    useEffect(() => {
        void loadClient(clientId);
    }, [clientId, loadClient]);

    const selectedSet = useMemo(() => new Set(selected), [selected]);
    const visible = useMemo(
        () => (client ? client.operations.filter((op) => operationMatches(op, query)) : []),
        [client, query],
    );
    const groups = useMemo(() => groupByTag(visible, t.apiClients.untagged), [visible, t.apiClients.untagged]);
    // Operations removed from the specification stay selected until unchecked; the node fails on them.
    const gone = client ? selected.filter((id) => !client.operations.some((op) => op.id === id)) : [];

    const toggle = (id: string) =>
        onChange(selectedSet.has(id) ? selected.filter((item) => item !== id) : [...selected, id]);

    const selectShown = () =>
        onChange([...selected, ...visible.filter((op) => !selectedSet.has(op.id)).map((op) => op.id)]);

    return (
        <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/60">
            <div className="flex items-center gap-2 px-2.5 py-2">
                <button
                    type="button"
                    onClick={() => setExpanded(!expanded)}
                    aria-expanded={expanded}
                    className="flex flex-1 min-w-0 items-center gap-2 text-left"
                >
                    <ChevronRight size={14} className={`shrink-0 text-slate-400 transition-transform ${expanded ? 'rotate-90' : ''}`}/>
                    <Plug size={14} className="shrink-0 text-indigo-500"/>
                    <span className="flex-1 min-w-0">
                        <span className="block truncate text-sm font-medium text-slate-700 dark:text-slate-200" title={client?.title}>
                            {client?.name ?? clientId}
                        </span>
                        {client && (
                            <span className="block text-[10px] text-slate-400">
                                {format(tt.selected, {count: selected.length, total: client.operations.length})}
                            </span>
                        )}
                    </span>
                </button>
                <button
                    type="button"
                    onClick={onRemove}
                    title={tt.remove}
                    aria-label={tt.remove}
                    className="p-1 rounded text-slate-400 hover:text-red-500 hover:bg-slate-100 dark:hover:bg-slate-700"
                >
                    <X size={14}/>
                </button>
            </div>

            {missing && (
                <p className="flex gap-1.5 px-3 pb-2 text-xs text-red-700 dark:text-red-300">
                    <AlertTriangle size={13} className="shrink-0 mt-0.5"/>
                    {format(tt.clientMissing, {id: clientId})}
                </p>
            )}
            {!missing && !client && (
                <div className="flex items-center gap-2 px-3 pb-2 text-xs text-slate-500">
                    <Loader2 size={13} className="animate-spin"/>
                    {tt.loading}
                </div>
            )}

            {client && expanded && (
                <div className="px-2.5 pb-2.5 space-y-2">
                    <div className="relative">
                        <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400"/>
                        <input
                            type="search"
                            value={query}
                            onChange={(e) => setQuery(e.target.value)}
                            placeholder={tt.filter}
                            aria-label={tt.filter}
                            className={`${configStyles.input} pl-8 py-1.5 text-xs`}
                        />
                    </div>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                        <button
                            type="button"
                            onClick={selectShown}
                            className="whitespace-nowrap text-[11px] font-medium text-blue-600 dark:text-blue-400 hover:underline"
                        >
                            {tt.selectAll}
                        </button>
                        <button
                            type="button"
                            onClick={() => onChange([])}
                            disabled={selected.length === 0}
                            className="whitespace-nowrap text-[11px] font-medium text-slate-500 hover:underline disabled:opacity-40 disabled:no-underline"
                        >
                            {tt.clear}
                        </button>
                    </div>

                    <div className="max-h-72 overflow-y-auto -mx-1">
                        {gone.map((id) => (
                            <label key={id} className="mx-1 flex items-start gap-2 px-1.5 py-1 rounded-md text-xs text-amber-700 dark:text-amber-300">
                                <input type="checkbox" checked onChange={() => toggle(id)} className={`${configStyles.checkbox} mt-0.5`}/>
                                <AlertTriangle size={12} className="shrink-0 mt-0.5"/>
                                <span className="min-w-0">
                                    <span className="block break-all font-mono">{id}</span>
                                    <span className="block text-[10px]">{tt.operationMissing}</span>
                                </span>
                            </label>
                        ))}
                        {groups.map(([tag, operations]) => (
                            <div key={tag}>
                                <p className="px-2.5 pt-1.5 pb-0.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400 truncate">{tag}</p>
                                {operations.map((op) => (
                                    <label
                                        key={op.id}
                                        title={`${op.method} ${op.path}${op.description ? `\n\n${op.description}` : ''}`}
                                        className="mx-1 flex items-center gap-2 px-1.5 py-1 rounded-md cursor-pointer select-none hover:bg-blue-50 dark:hover:bg-slate-700/60"
                                    >
                                        <input
                                            type="checkbox"
                                            checked={selectedSet.has(op.id)}
                                            onChange={() => toggle(op.id)}
                                            className={configStyles.checkbox}
                                        />
                                        <MethodBadge method={op.method}/>
                                        <span className={`flex-1 truncate text-xs ${op.deprecated ? 'line-through text-slate-400' : 'text-slate-600 dark:text-slate-300'}`}>
                                            {operationLabel(op)}
                                        </span>
                                    </label>
                                ))}
                            </div>
                        ))}
                        {visible.length === 0 && (
                            <p className="px-2.5 py-2 text-xs text-slate-500 dark:text-slate-400">{tt.noMatches}</p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
};
