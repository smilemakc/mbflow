import React, {useEffect} from 'react';
import {AlertTriangle} from 'lucide-react';
import {useApiClientStore} from '@/store/apiClientStore';
import {useTranslation} from '@/store/translations';
import {MethodBadge} from '@/components/apiClients/MethodBadge';
import {format} from '@/components/apiClients/schemaUtils';

interface OpenAPINodeSummaryProps {
    config?: Record<string, any>;
}

/** Method and path of the operation an "openapi" node calls, shown on the canvas. */
export const OpenAPINodeSummary: React.FC<OpenAPINodeSummaryProps> = ({config}) => {
    const t = useTranslation();
    const clientId: string | undefined = config?.client_id;
    const operationId: string | undefined = config?.operation_id;
    const client = useApiClientStore((s) => (clientId ? s.details[clientId] : undefined));
    const missing = useApiClientStore((s) => (clientId ? s.missing[clientId] : false));
    const loadClient = useApiClientStore((s) => s.loadClient);

    useEffect(() => {
        if (clientId) void loadClient(clientId);
    }, [clientId, loadClient]);

    if (!clientId || !operationId) return null;
    const operation = client?.operations.find((op) => op.id === operationId);

    if (!operation) {
        if (!client && !missing) return null;
        const message = missing ? t.nodeConfig.openapi.clientMissing : format(t.nodeConfig.openapi.operationMissing, {id: operationId});
        return (
            <div className="mt-2 flex items-center gap-1 text-[10px] text-amber-600 dark:text-amber-400" title={message}>
                <AlertTriangle size={11} className="shrink-0"/>
                <span className="font-mono truncate">{operationId}</span>
            </div>
        );
    }

    return (
        <div className="mt-2 flex items-center gap-1.5 min-w-0" title={`${operation.method} ${operation.path}`}>
            <MethodBadge method={operation.method}/>
            <span className="font-mono text-[10px] text-slate-500 dark:text-slate-400 truncate">{operation.path}</span>
        </div>
    );
};
