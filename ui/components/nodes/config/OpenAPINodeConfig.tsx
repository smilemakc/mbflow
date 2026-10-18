import React, {useEffect, useMemo, useState} from 'react';
import {Link} from 'react-router-dom';
import {AlertTriangle, ExternalLink, KeyRound, Loader2, Unlock, Wand2} from 'lucide-react';
import {useTranslation} from '@/store/translations';
import {useApiClientStore} from '@/store/apiClientStore';
import {configStyles} from '@/styles/configStyles';
import {VariableAutocomplete} from '@/components/builder/VariableAutocomplete';
import {MethodBadge} from '@/components/apiClients/MethodBadge';
import {KeyValueRows} from '@/components/apiClients/KeyValueRows';
import {
    buildExample,
    describeSchema,
    flattenSchema,
    format,
    groupByTag,
    objectFields,
    schemaType,
    SchemaField,
    successResponse,
} from '@/components/apiClients/schemaUtils';
import type {ApiClient, ApiOperation, ApiParameter, JsonSchema, ParameterLocation} from '@/services/apiClientService';

interface OpenAPINodeConfigProps {
    config: Record<string, any>;
    nodeId?: string;
    onChange: (config: Record<string, any>) => void;
}

type ValueMap = Record<string, unknown>;

const PARAM_CONFIG_KEYS: Record<ParameterLocation, string> = {
    path: 'path_params',
    query: 'query_params',
    header: 'headers',
    cookie: 'cookies',
};

/** Config keys kept when another operation is selected. */
const OPERATION_INDEPENDENT_KEYS = ['client_id', 'base_url', 'response_type', 'ignore_status_errors', 'success_status_codes', 'description'];

const isObject = (value: unknown): value is ValueMap =>
    typeof value === 'object' && value !== null && !Array.isArray(value);

const asMap = (value: unknown): ValueMap => (isObject(value) ? value : {});

const asText = (value: unknown): string => {
    if (value === undefined || value === null) return '';
    if (typeof value === 'string') return value;
    return JSON.stringify(value);
};

/** Values of declared parameters are edited in their own fields; the rest are "extra". */
const splitDeclared = (values: ValueMap, declared: ApiParameter[], caseInsensitive: boolean) => {
    const names = new Set(declared.map((p) => (caseInsensitive ? p.name.toLowerCase() : p.name)));
    const extra: Record<string, string> = {};
    Object.entries(values).forEach(([key, value]) => {
        if (!names.has(caseInsensitive ? key.toLowerCase() : key)) extra[key] = asText(value);
    });
    return extra;
};

/** Converts a field value typed in the form into the JSON value its schema declares. */
const toTypedValue = (value: unknown, schema: JsonSchema | undefined, schemas: Record<string, JsonSchema>): unknown => {
    if (typeof value !== 'string') return value;
    const trimmed = value.trim();
    switch (schemaType(flattenSchema(schema, schemas))) {
        case 'integer':
        case 'number':
            return trimmed !== '' && !Number.isNaN(Number(trimmed)) ? Number(trimmed) : value;
        case 'boolean':
            return trimmed === 'true' ? true : trimmed === 'false' ? false : value;
        case 'object':
        case 'array':
            try {
                return JSON.parse(trimmed);
            } catch {
                return value;
            }
        default:
            return value;
    }
};

export const OpenAPINodeConfigComponent: React.FC<OpenAPINodeConfigProps> = ({config, nodeId, onChange}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.openapi;
    const clientId: string | undefined = config.client_id;
    const operationId: string | undefined = config.operation_id;
    const client = useApiClientStore((s) => (clientId ? s.details[clientId] : undefined));
    const missing = useApiClientStore((s) => (clientId ? s.missing[clientId] : false));
    const loadClient = useApiClientStore((s) => s.loadClient);

    useEffect(() => {
        if (clientId) void loadClient(clientId);
    }, [clientId, loadClient]);

    const operation = client?.operations.find((op) => op.id === operationId);

    const update = (patch: ValueMap) => {
        const next: ValueMap = {...config, ...patch};
        Object.keys(patch).forEach((key) => {
            const value = next[key];
            if (value === undefined || value === '' || (isObject(value) && Object.keys(value).length === 0)) {
                delete next[key];
            }
        });
        onChange(next);
    };

    // Values of parameters the new operation also declares are kept, so
    // re-selecting a renamed operation does not lose the node settings.
    const selectOperation = (id: string) => {
        const target = client?.operations.find((op) => op.id === id);
        const next: ValueMap = {};
        OPERATION_INDEPENDENT_KEYS.forEach((key) => {
            if (config[key] !== undefined) next[key] = config[key];
        });
        next.operation_id = id;
        if (target) {
            (Object.keys(PARAM_CONFIG_KEYS) as ParameterLocation[]).forEach((location) => {
                const normalize = (name: string) => (location === 'header' ? name.toLowerCase() : name);
                const declared = new Set((target.parameters ?? []).filter((p) => p.in === location).map((p) => normalize(p.name)));
                const kept = Object.fromEntries(
                    Object.entries(asMap(config[PARAM_CONFIG_KEYS[location]])).filter(([name]) => declared.has(normalize(name))),
                );
                if (Object.keys(kept).length > 0) next[PARAM_CONFIG_KEYS[location]] = kept;
            });
            if (target.request_body && config.body !== undefined) next.body = config.body;
        }
        onChange(next);
    };

    if (!clientId || missing) {
        return <Notice tone="error">{tt.clientMissing}</Notice>;
    }
    if (!client) {
        return (
            <div className="flex items-center gap-2 text-xs text-slate-500">
                <Loader2 size={14} className="animate-spin"/>
                {tt.loading}
            </div>
        );
    }

    return (
        <div className="space-y-5">
            <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 truncate" title={client.title}>
                        {client.name}
                    </span>
                    <Link
                        to="/api-clients"
                        className="inline-flex items-center gap-0.5 text-[11px] text-blue-600 dark:text-blue-400 hover:underline shrink-0"
                    >
                        {tt.openClient}
                        <ExternalLink size={10}/>
                    </Link>
                </div>
                <OperationSelect client={client} value={operationId ?? ''} onChange={selectOperation}/>
            </div>

            {!operation ? (
                <Notice tone="warning">{format(tt.operationMissing, {id: operationId ?? ''})}</Notice>
            ) : (
                <>
                    <OperationHeader client={client} operation={operation}/>
                    {(['path', 'query', 'header', 'cookie'] as ParameterLocation[]).map((location) => (
                        <ParameterSection
                            key={location}
                            location={location}
                            client={client}
                            operation={operation}
                            values={asMap(config[PARAM_CONFIG_KEYS[location]])}
                            onChange={(values) => update({[PARAM_CONFIG_KEYS[location]]: values})}
                        />
                    ))}
                    {operation.request_body && (
                        <BodySection
                            key={`${nodeId}:${operation.id}`}
                            client={client}
                            operation={operation}
                            body={config.body}
                            contentType={config.content_type}
                            onChange={update}
                        />
                    )}
                    <AdvancedSection
                        key={`${nodeId}:advanced`}
                        client={client}
                        operation={operation}
                        config={config}
                        nodeId={nodeId}
                        onChange={update}
                    />
                    <OutputHint client={client} operation={operation}/>
                </>
            )}
        </div>
    );
};

const Notice: React.FC<{tone: 'error' | 'warning'; children: React.ReactNode}> = ({tone, children}) => (
    <div className={`${configStyles.infoBox[tone]} flex gap-2 text-xs ${tone === 'error' ? 'text-red-700 dark:text-red-300' : 'text-amber-800 dark:text-amber-300'}`}>
        <AlertTriangle size={14} className="shrink-0 mt-0.5"/>
        <span>{children}</span>
    </div>
);

const OperationSelect: React.FC<{client: ApiClient; value: string; onChange: (id: string) => void}> = ({client, value, onChange}) => {
    const t = useTranslation();
    const groups = useMemo(() => groupByTag(client.operations, t.apiClients.untagged), [client.operations, t.apiClients.untagged]);
    return (
        <select value={value} onChange={(e) => onChange(e.target.value)} className={`${configStyles.select} text-xs`} aria-label={t.nodeConfig.openapi.operation}>
            {!client.operations.some((op) => op.id === value) && <option value={value}>{value}</option>}
            {groups.map(([tag, operations]) => (
                <optgroup key={tag} label={tag}>
                    {operations.map((op) => (
                        <option key={op.id} value={op.id}>
                            {op.method} {op.summary || op.path}
                        </option>
                    ))}
                </optgroup>
            ))}
        </select>
    );
};

const OperationHeader: React.FC<{client: ApiClient; operation: ApiOperation}> = ({client, operation}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.openapi;
    const [expanded, setExpanded] = useState(false);
    const authType = client.auth?.type && client.auth.type !== 'none' ? client.auth.type : null;

    return (
        <div className={`rounded-lg p-3 space-y-2 ${configStyles.gradientHeader.blue}`}>
            <div className="flex items-start gap-2 min-w-0">
                <MethodBadge method={operation.method}/>
                <code className="text-xs font-mono text-slate-700 dark:text-slate-200 break-all">{operation.path}</code>
            </div>
            {operation.description && (
                <button
                    type="button"
                    onClick={() => setExpanded(!expanded)}
                    className={`block w-full text-left text-xs text-slate-600 dark:text-slate-300 whitespace-pre-line ${expanded ? '' : 'line-clamp-3'}`}
                >
                    {operation.description}
                </button>
            )}
            {operation.deprecated && (
                <p className="flex items-center gap-1 text-[11px] text-amber-600"><AlertTriangle size={11}/>{t.apiClients.deprecated}</p>
            )}
            {authType && (
                <p className="flex items-center gap-1 text-[11px] text-slate-500 dark:text-slate-400">
                    {operation.no_auth ? <Unlock size={11}/> : <KeyRound size={11}/>}
                    {operation.no_auth ? tt.noAuth : format(tt.authLabel, {type: t.apiClients.authTypes[authType]})}
                </p>
            )}
        </div>
    );
};

const sectionTitle = (location: ParameterLocation, tt: Record<string, any>): string => ({
    path: tt.pathParams,
    query: tt.queryParams,
    header: tt.headerParams,
    cookie: tt.cookieParams,
})[location];

const ParameterSection: React.FC<{
    location: ParameterLocation;
    client: ApiClient;
    operation: ApiOperation;
    values: ValueMap;
    onChange: (values: ValueMap) => void;
}> = ({location, client, operation, values, onChange}) => {
    const t = useTranslation();
    const params = (operation.parameters ?? []).filter((p) => p.in === location);
    if (params.length === 0) return null;

    const setValue = (name: string, value: string) => {
        const next = {...values};
        const existing = location === 'header'
            ? Object.keys(next).find((key) => key.toLowerCase() === name.toLowerCase())
            : name;
        if (existing) delete next[existing];
        if (value !== '') next[name] = value;
        onChange(next);
    };

    const getValue = (name: string): unknown => {
        if (location !== 'header') return values[name];
        const key = Object.keys(values).find((k) => k.toLowerCase() === name.toLowerCase());
        return key ? values[key] : undefined;
    };

    return (
        <section className="space-y-3">
            <h4 className={configStyles.sectionTitle}>{sectionTitle(location, t.nodeConfig.openapi)}</h4>
            {params.map((param) => (
                <ValueField
                    key={param.name}
                    name={param.name}
                    required={!!param.required}
                    schema={param.schema}
                    description={param.description}
                    example={param.example}
                    schemas={client.schemas}
                    value={getValue(param.name)}
                    onChange={(value) => setValue(param.name, value)}
                />
            ))}
        </section>
    );
};

/** A labelled input for a parameter or a body field, with type hint and quick values. */
const ValueField: React.FC<{
    name: string;
    required: boolean;
    schema?: JsonSchema;
    description?: string;
    example?: unknown;
    schemas: Record<string, JsonSchema>;
    value: unknown;
    onChange: (value: string) => void;
}> = ({name, required, schema, description, example, schemas, value, onChange}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.openapi;
    const resolved = flattenSchema(schema, schemas);
    const type = schemaType(resolved);
    const quickValues: string[] = Array.isArray(resolved?.enum)
        ? resolved!.enum.slice(0, 12).map((v: unknown) => asText(v))
        : type === 'boolean' ? ['true', 'false'] : [];
    const multiline = type === 'object' || type === 'array';
    const sample = example ?? resolved?.example ?? resolved?.default;
    const placeholder = sample !== undefined ? asText(sample) : required ? tt.required : '';
    const text = asText(value);

    return (
        <div className="space-y-1">
            <div className="flex items-baseline justify-between gap-2">
                <label className="text-xs font-semibold text-slate-700 dark:text-slate-300 font-mono truncate" title={name}>
                    {name}{required && <span className={configStyles.labelRequired}>*</span>}
                </label>
                <span className="text-[10px] text-slate-400 font-mono truncate max-w-[55%]" title={describeSchema(schema, schemas)}>
                    {describeSchema(schema, schemas)}
                </span>
            </div>
            <VariableAutocomplete
                value={text}
                onChange={onChange}
                placeholder={placeholder}
                type={multiline ? 'textarea' : 'input'}
                rows={multiline ? 3 : undefined}
                className={`${multiline ? configStyles.textareaMonospace : configStyles.input} text-xs ${required && !text ? 'border-amber-300 dark:border-amber-700' : ''}`}
            />
            {quickValues.length > 0 && (
                <div className="flex flex-wrap items-center gap-1">
                    <span className="text-[10px] text-slate-400">{tt.options}</span>
                    {quickValues.map((option) => (
                        <button
                            key={option}
                            type="button"
                            onClick={() => onChange(option)}
                            className={`px-1.5 py-0.5 rounded text-[10px] font-mono border transition-colors ${
                                text === option
                                    ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 hover:border-blue-400'
                            }`}
                        >
                            {option}
                        </button>
                    ))}
                </div>
            )}
            {description && <p className="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-3" title={description}>{description}</p>}
        </div>
    );
};

const BodySection: React.FC<{
    client: ApiClient;
    operation: ApiOperation;
    body: unknown;
    contentType?: string;
    onChange: (patch: ValueMap) => void;
}> = ({client, operation, body, contentType, onChange}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.openapi;
    const content = operation.request_body!.content;
    const media = content.find((m) => m.content_type === contentType) ?? content[0];
    const fields: SchemaField[] | null = useMemo(() => objectFields(media?.schema, client.schemas), [media, client.schemas]);
    const mode: 'fields' | 'raw' = fields && typeof body !== 'string' ? 'fields' : 'raw';
    const [modeError, setModeError] = useState(false);
    const values = asMap(body);

    const switchMode = (target: 'fields' | 'raw') => {
        setModeError(false);
        if (target === mode) return;
        if (target === 'raw') {
            const typed: ValueMap = {};
            Object.entries(values).forEach(([key, value]) => {
                const field = fields?.find((f) => f.name === key);
                typed[key] = toTypedValue(value, field?.schema, client.schemas);
            });
            onChange({body: Object.keys(typed).length > 0 ? JSON.stringify(typed, null, 2) : undefined});
            return;
        }
        try {
            const parsed = typeof body === 'string' && body.trim() ? JSON.parse(body) : {};
            if (!isObject(parsed)) throw new Error('not an object');
            const textValues: ValueMap = {};
            Object.entries(parsed).forEach(([key, value]) => {
                textValues[key] = asText(value);
            });
            onChange({body: textValues});
        } catch {
            setModeError(true);
        }
    };

    const insertExample = () => {
        const example = media?.example ?? buildExample(media?.schema, client.schemas);
        if (mode === 'fields' && isObject(example)) {
            const textValues: ValueMap = {};
            Object.entries(example).forEach(([key, value]) => {
                textValues[key] = asText(value);
            });
            onChange({body: {...textValues, ...values}});
        } else {
            onChange({body: typeof example === 'string' ? example : JSON.stringify(example ?? {}, null, 2)});
        }
    };

    const setField = (name: string, value: string) => {
        const next = {...values};
        if (value === '') delete next[name];
        else next[name] = value;
        onChange({body: next});
    };

    return (
        <section className="space-y-3">
            <div className="flex items-center justify-between gap-2">
                <h4 className={configStyles.sectionTitle}>
                    {tt.body}{operation.request_body!.required && <span className={configStyles.labelRequired}>*</span>}
                </h4>
                <button
                    type="button"
                    onClick={insertExample}
                    className="inline-flex items-center gap-1 text-[11px] font-medium text-blue-600 dark:text-blue-400 hover:underline"
                >
                    <Wand2 size={12}/>
                    {tt.insertExample}
                </button>
            </div>

            {content.length > 1 && (
                <select
                    value={media?.content_type}
                    onChange={(e) => onChange({content_type: e.target.value === content[0].content_type ? undefined : e.target.value})}
                    className={`${configStyles.select} text-xs font-mono`}
                    aria-label={tt.contentType}
                >
                    {content.map((m) => <option key={m.content_type} value={m.content_type}>{m.content_type}</option>)}
                </select>
            )}

            {fields && (
                <div className="inline-flex rounded-lg border border-slate-200 dark:border-slate-700 p-0.5 text-[11px]" role="group">
                    {(['fields', 'raw'] as const).map((value) => (
                        <button
                            key={value}
                            type="button"
                            aria-pressed={mode === value}
                            onClick={() => switchMode(value)}
                            className={`px-2.5 py-1 rounded-md font-medium transition-colors ${
                                mode === value
                                    ? 'bg-blue-600 text-white'
                                    : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
                            }`}
                        >
                            {value === 'fields' ? tt.fieldsMode : tt.rawMode}
                        </button>
                    ))}
                </div>
            )}
            {modeError && <Notice tone="warning">{tt.bodyNotObject}</Notice>}

            {mode === 'fields' && fields ? (
                <>
                    {fields.map((field) => (
                        <ValueField
                            key={field.name}
                            name={field.name}
                            required={field.required}
                            schema={field.schema}
                            description={flattenSchema(field.schema, client.schemas)?.description}
                            schemas={client.schemas}
                            value={values[field.name]}
                            onChange={(value) => setField(field.name, value)}
                        />
                    ))}
                    <p className={configStyles.hint}>{tt.bodyHint}</p>
                </>
            ) : (
                <>
                    <VariableAutocomplete
                        value={asText(body)}
                        onChange={(value) => onChange({body: value})}
                        type="textarea"
                        rows={8}
                        placeholder={media?.schema ? JSON.stringify(buildExample(media.schema, client.schemas), null, 2) : ''}
                        className={`${configStyles.textareaMonospace} text-xs`}
                    />
                    <p className={configStyles.hint}>{tt.rawHint}</p>
                </>
            )}
        </section>
    );
};

const AdvancedSection: React.FC<{
    client: ApiClient;
    operation: ApiOperation;
    config: Record<string, any>;
    nodeId?: string;
    onChange: (patch: ValueMap) => void;
}> = ({client, operation, config, nodeId, onChange}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.openapi;
    const declared = operation.parameters ?? [];
    const headers = asMap(config.headers);
    const query = asMap(config.query_params);
    const extraHeaders = splitDeclared(headers, declared.filter((p) => p.in === 'header'), true);
    const extraQuery = splitDeclared(query, declared.filter((p) => p.in === 'query'), false);

    const mergeExtras = (current: ValueMap, previousExtras: Record<string, string>, extras: Record<string, string>) => {
        const next = {...current};
        Object.keys(previousExtras).forEach((key) => delete next[key]);
        return {...next, ...extras};
    };

    return (
        <details className="group rounded-lg border border-slate-200 dark:border-slate-700">
            <summary className="cursor-pointer select-none px-3 py-2 text-xs font-semibold text-slate-600 dark:text-slate-300">
                {tt.advanced}
            </summary>
            <div className="px-3 pb-3 space-y-4">
                <div className="space-y-1">
                    <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">{tt.baseUrlOverride}</label>
                    <VariableAutocomplete
                        value={asText(config.base_url)}
                        onChange={(value) => onChange({base_url: value.trim()})}
                        placeholder={client.base_url || 'https://api.example.com'}
                        className={`${configStyles.input} text-xs font-mono`}
                    />
                    <p className={configStyles.hint}>{format(tt.baseUrlOverrideHint, {url: client.base_url || '—'})}</p>
                </div>
                <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">{tt.extraHeaders}</label>
                    <KeyValueRows
                        value={extraHeaders}
                        onChange={(extras) => onChange({headers: mergeExtras(headers, extraHeaders, extras)})}
                        addLabel={t.apiClients.addHeader}
                        keyPlaceholder="X-Header"
                        valuePlaceholder="{{input.value}}"
                        enableVariables
                        resetKey={`${nodeId}:${operation.id}:headers`}
                    />
                </div>
                <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">{tt.extraQuery}</label>
                    <KeyValueRows
                        value={extraQuery}
                        onChange={(extras) => onChange({query_params: mergeExtras(query, extraQuery, extras)})}
                        addLabel={tt.addParam}
                        keyPlaceholder="param"
                        valuePlaceholder="{{input.value}}"
                        enableVariables
                        resetKey={`${nodeId}:${operation.id}:query`}
                    />
                </div>
                <div className="space-y-1">
                    <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">{tt.responseType}</label>
                    <select
                        value={config.response_type ?? 'auto'}
                        onChange={(e) => onChange({response_type: e.target.value === 'auto' ? undefined : e.target.value})}
                        className={`${configStyles.select} text-xs`}
                    >
                        <option value="auto">{tt.responseAuto}</option>
                        <option value="binary">{tt.responseBinary}</option>
                    </select>
                </div>
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={config.ignore_status_errors === true}
                        onChange={(e) => onChange({ignore_status_errors: e.target.checked ? true : undefined})}
                        className={configStyles.checkbox}
                    />
                    <span className="text-xs text-slate-700 dark:text-slate-300">{tt.ignoreStatusErrors}</span>
                </label>
            </div>
        </details>
    );
};

const OutputHint: React.FC<{client: ApiClient; operation: ApiOperation}> = ({client, operation}) => {
    const t = useTranslation();
    const tt = t.nodeConfig.openapi;
    const response = successResponse(operation);
    const schema = response?.content?.[0]?.schema;
    return (
        <div className={`${configStyles.infoBox.info} space-y-1 text-[11px] text-blue-800 dark:text-blue-300`}>
            <p className="font-semibold">{tt.output}</p>
            {response && schema && (
                <p className="font-mono break-all">
                    {format(tt.responseSchema, {status: response.status, schema: describeSchema(schema, client.schemas)})}
                </p>
            )}
            <p>{tt.outputHint}</p>
        </div>
    );
};
