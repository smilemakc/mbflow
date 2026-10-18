import React, {useEffect, useMemo, useState} from 'react';
import {ChevronRight, GripVertical, Loader2, Plug, Plus, Settings2} from 'lucide-react';
import {useNavigate} from 'react-router-dom';
import {NodeType} from '@/types.ts';
import {useTranslation} from '@/store/translations.ts';
import {useApiClientStore} from '@/store/apiClientStore';
import {MethodBadge} from '@/components/apiClients/MethodBadge';
import {groupByTag, operationLabel, operationMatches} from '@/components/apiClients/schemaUtils';
import type {ApiClient, ApiOperation} from '@/services/apiClientService';
import {NODE_TEMPLATE_MIME, NODE_TYPE_MIME, NodeTemplate} from './dragData';

interface ApiOperationsLibraryProps {
    searchQuery: string;
    categoryLabel: string;
}

const operationTemplate = (client: ApiClient, operation: ApiOperation): NodeTemplate => ({
    label: operationLabel(operation),
    description: client.name,
    config: {client_id: client.id, operation_id: operation.id},
});

/** Node library section with a draggable node for every operation of every imported API client. */
export const ApiOperationsLibrary: React.FC<ApiOperationsLibraryProps> = ({searchQuery, categoryLabel}) => {
    const t = useTranslation();
    const navigate = useNavigate();
    const {clients, details, isLoaded, loadAllDetails} = useApiClientStore();
    const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

    useEffect(() => {
        void loadAllDetails();
    }, [loadAllDetails]);

    const query = searchQuery.trim();
    const visible = useMemo(() => clients
        .map((summary) => {
            const client = details[summary.id];
            const operations = client ? client.operations.filter((op) => operationMatches(op, query)) : [];
            const clientMatches = !query || summary.name.toLowerCase().includes(query.toLowerCase());
            return {summary, client, operations: clientMatches && client ? client.operations : operations};
        })
        .filter(({client, operations}) => !query || (client && operations.length > 0)),
    [clients, details, query]);

    if (query && visible.length === 0) return null;

    const onDragStart = (event: React.DragEvent, client: ApiClient, operation: ApiOperation) => {
        event.dataTransfer.setData(NODE_TYPE_MIME, NodeType.OPENAPI);
        event.dataTransfer.setData(NODE_TEMPLATE_MIME, JSON.stringify(operationTemplate(client, operation)));
        event.dataTransfer.effectAllowed = 'move';
    };

    return (
        <div>
            <div className="flex items-center justify-between mb-2 ml-1">
                <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400">{categoryLabel}</h3>
                <button
                    type="button"
                    onClick={() => navigate('/api-clients')}
                    title={t.builder.manageApis}
                    className="p-1 rounded text-slate-400 hover:text-blue-500 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                    <Settings2 size={14}/>
                </button>
            </div>

            {!isLoaded && (
                <div className="flex justify-center py-3"><Loader2 size={16} className="animate-spin text-slate-400"/></div>
            )}

            {isLoaded && clients.length === 0 && (
                <div className="p-3 rounded-lg border border-dashed border-slate-300 dark:border-slate-700 text-xs text-slate-500 dark:text-slate-400 space-y-2">
                    <p>{t.builder.noApis}</p>
                    <button
                        type="button"
                        onClick={() => navigate('/api-clients')}
                        className="inline-flex items-center gap-1 font-medium text-blue-600 dark:text-blue-400 hover:underline"
                    >
                        <Plus size={12}/>
                        {t.builder.importApi}
                    </button>
                </div>
            )}

            <div className="space-y-1.5">
                {visible.map(({summary, client, operations}) => {
                    const isCollapsed = !query && (collapsed[summary.id] ?? clients.length > 1);
                    return (
                        <div key={summary.id} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/60">
                            <button
                                type="button"
                                onClick={() => setCollapsed({...collapsed, [summary.id]: !isCollapsed})}
                                aria-expanded={!isCollapsed}
                                className="w-full flex items-center gap-2 px-2.5 py-2 text-left"
                            >
                                <ChevronRight size={14} className={`shrink-0 text-slate-400 transition-transform ${isCollapsed ? '' : 'rotate-90'}`}/>
                                <Plug size={14} className="shrink-0 text-indigo-500"/>
                                <span className="flex-1 truncate text-sm font-medium text-slate-700 dark:text-slate-200" title={summary.title}>
                                    {summary.name}
                                </span>
                                <span className="text-[10px] text-slate-400">{summary.operation_count}</span>
                            </button>
                            {!isCollapsed && (
                                <div className="pb-1.5 max-h-80 overflow-y-auto">
                                    {!client && (
                                        <div className="flex justify-center py-2"><Loader2 size={14} className="animate-spin text-slate-400"/></div>
                                    )}
                                    {client && groupByTag(operations, t.apiClients.untagged).map(([tag, ops]) => (
                                        <div key={tag}>
                                            <p className="px-3 pt-1.5 pb-0.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400 truncate">{tag}</p>
                                            {ops.map((operation) => (
                                                <div
                                                    key={operation.id}
                                                    draggable
                                                    onDragStart={(event) => onDragStart(event, client, operation)}
                                                    title={`${operation.method} ${operation.path}${operation.description ? `\n\n${operation.description}` : ''}`}
                                                    className="group mx-1.5 flex items-center gap-2 px-1.5 py-1 rounded-md cursor-grab active:cursor-grabbing hover:bg-blue-50 dark:hover:bg-slate-700/60 select-none"
                                                >
                                                    <MethodBadge method={operation.method}/>
                                                    <span className={`flex-1 truncate text-xs ${operation.deprecated ? 'line-through text-slate-400' : 'text-slate-600 dark:text-slate-300'}`}>
                                                        {operationLabel(operation)}
                                                    </span>
                                                    <GripVertical size={12} className="shrink-0 text-slate-300 opacity-0 group-hover:opacity-100"/>
                                                </div>
                                            ))}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
        </div>
    );
};
