import React, {useEffect, useMemo, useState} from 'react';
import {ExternalLink, KeyRound} from 'lucide-react';
import {Link} from 'react-router-dom';
import {FormField, Select} from '@/components/ui/form';
import {configStyles} from '@/styles/configStyles';
import {useTranslation} from '@/store/translations';
import {Credential, credentialsApi, CredentialType} from '@/services/credentialsService';
import type {ApiAuthType, ApiClientAuth, ApiKeyLocation, ApiSecurityScheme} from '@/services/apiClientService';
import {format} from './schemaUtils';

/** Credential types holding the secret each auth type needs; mirrors the backend check. */
const COMPATIBLE_CREDENTIALS: Record<ApiAuthType, CredentialType[]> = {
    none: [],
    api_key: ['api_key', 'oauth2', 'custom'],
    bearer: ['api_key', 'oauth2', 'custom'],
    basic: ['basic_auth', 'custom'],
    oauth2_client_credentials: ['oauth2', 'custom'],
};

const AUTH_TYPES: ApiAuthType[] = ['none', 'api_key', 'bearer', 'basic', 'oauth2_client_credentials'];
const KEY_LOCATIONS: ApiKeyLocation[] = ['header', 'query', 'cookie'];

/** Auth settings for a security scheme of the specification; null when unsupported. */
export const authFromScheme = (scheme: ApiSecurityScheme): ApiClientAuth | null => {
    switch (scheme.type.toLowerCase()) {
        case 'apikey':
            if (!scheme.param_name || !['header', 'query', 'cookie'].includes(scheme.in ?? '')) return null;
            return {type: 'api_key', in: scheme.in as ApiKeyLocation, name: scheme.param_name, scheme_name: scheme.name};
        case 'http':
            if (scheme.scheme === 'basic') return {type: 'basic', scheme_name: scheme.name};
            if (scheme.scheme === 'bearer') return {type: 'bearer', scheme_name: scheme.name};
            return null;
        case 'oauth2':
            return scheme.token_url
                ? {type: 'oauth2_client_credentials', token_url: scheme.token_url, scopes: scheme.scopes, scheme_name: scheme.name}
                : {type: 'bearer', scheme_name: scheme.name};
        case 'openidconnect':
            return {type: 'bearer', scheme_name: scheme.name};
        default:
            return null;
    }
};

interface AuthSettingsFormProps {
    value: ApiClientAuth;
    onChange: (auth: ApiClientAuth) => void;
    securitySchemes?: ApiSecurityScheme[];
}

export const AuthSettingsForm: React.FC<AuthSettingsFormProps> = ({value, onChange, securitySchemes = []}) => {
    const t = useTranslation();
    const [credentials, setCredentials] = useState<Credential[]>([]);
    const [credentialsUnavailable, setCredentialsUnavailable] = useState(false);

    useEffect(() => {
        let cancelled = false;
        credentialsApi.listCredentials()
            .then((response) => {
                if (!cancelled) setCredentials(response.data.credentials ?? []);
            })
            .catch(() => {
                if (!cancelled) setCredentialsUnavailable(true);
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const compatible = useMemo(
        () => credentials.filter((c) => COMPATIBLE_CREDENTIALS[value.type]?.includes(c.credential_type)),
        [credentials, value.type],
    );

    const applicableSchemes = useMemo(
        () => securitySchemes
            .map((scheme) => ({scheme, auth: authFromScheme(scheme)}))
            .filter((entry): entry is {scheme: ApiSecurityScheme; auth: ApiClientAuth} => entry.auth !== null),
        [securitySchemes],
    );

    const changeType = (type: ApiAuthType) => {
        const keep = credentials.find((c) => c.id === value.credential_id);
        const credentialStillFits = keep && COMPATIBLE_CREDENTIALS[type].includes(keep.credential_type);
        onChange({
            type,
            credential_id: credentialStillFits ? value.credential_id : undefined,
            in: type === 'api_key' ? value.in ?? 'header' : undefined,
            name: type === 'api_key' ? value.name ?? 'X-API-Key' : undefined,
            token_url: type === 'oauth2_client_credentials' ? value.token_url : undefined,
            scopes: type === 'oauth2_client_credentials' ? value.scopes : undefined,
        });
    };

    return (
        <div className="space-y-3">
            {applicableSchemes.length > 0 && (
                <div className="flex flex-wrap gap-1.5">
                    {applicableSchemes.map(({scheme, auth}) => {
                        const active = value.scheme_name === scheme.name && value.type === auth.type;
                        return (
                            <button
                                key={scheme.name}
                                type="button"
                                onClick={() => onChange({...auth, credential_id: value.credential_id})}
                                className={`inline-flex items-center gap-1 px-2 py-1 rounded-md border text-xs transition-colors ${
                                    active
                                        ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                                        : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 hover:border-blue-400'
                                }`}
                                title={scheme.description}
                            >
                                <KeyRound size={12}/>
                                {format(t.apiClients.schemeFromSpec, {scheme: scheme.name})}
                            </button>
                        );
                    })}
                </div>
            )}

            <FormField label={t.apiClients.authType}>
                <Select
                    value={value.type || 'none'}
                    onChange={(v) => changeType(v as ApiAuthType)}
                    options={AUTH_TYPES.map((type) => ({value: type, label: t.apiClients.authTypes[type]}))}
                />
            </FormField>

            {value.type === 'api_key' && (
                <div className="grid grid-cols-2 gap-2">
                    <FormField label={t.apiClients.keyLocation}>
                        <Select
                            value={value.in ?? 'header'}
                            onChange={(v) => onChange({...value, in: v as ApiKeyLocation})}
                            options={KEY_LOCATIONS.map((loc) => ({value: loc, label: t.apiClients.locations[loc]}))}
                        />
                    </FormField>
                    <FormField label={t.apiClients.keyName}>
                        <input
                            type="text"
                            value={value.name ?? ''}
                            onChange={(e) => onChange({...value, name: e.target.value})}
                            className={`${configStyles.input} font-mono`}
                            placeholder="X-API-Key"
                        />
                    </FormField>
                </div>
            )}

            {value.type === 'oauth2_client_credentials' && (
                <>
                    <FormField label={t.apiClients.tokenUrl} hint={t.apiClients.tokenUrlHint}>
                        <input
                            type="url"
                            value={value.token_url ?? ''}
                            onChange={(e) => onChange({...value, token_url: e.target.value || undefined})}
                            className={configStyles.input}
                            placeholder="https://auth.example.com/oauth/token"
                        />
                    </FormField>
                    <FormField label={t.apiClients.scopes}>
                        <input
                            type="text"
                            value={(value.scopes ?? []).join(' ')}
                            onChange={(e) => {
                                const scopes = e.target.value.split(/\s+/).filter(Boolean);
                                onChange({...value, scopes: scopes.length > 0 ? scopes : undefined});
                            }}
                            className={`${configStyles.input} font-mono`}
                            placeholder="read write"
                        />
                    </FormField>
                </>
            )}

            {value.type !== 'none' && (
                <FormField label={t.apiClients.credential} hint={credentialsUnavailable ? undefined : t.apiClients.credentialHint}>
                    {credentialsUnavailable ? (
                        <p className="text-xs text-amber-600 dark:text-amber-400">{t.apiClients.credentialsUnavailable}</p>
                    ) : (
                        <>
                            <Select
                                value={value.credential_id ?? ''}
                                onChange={(v) => onChange({...value, credential_id: v || undefined})}
                                options={[
                                    {value: '', label: t.apiClients.credentialNone},
                                    ...compatible.map((c) => ({value: c.id, label: `${c.name} (${c.credential_type})`})),
                                ]}
                            />
                            {compatible.length === 0 && (
                                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                    {t.apiClients.noCompatibleCredentials}{' '}
                                    <Link to="/resources" className="inline-flex items-center gap-0.5 text-blue-600 dark:text-blue-400 hover:underline">
                                        {t.apiClients.createCredential}
                                        <ExternalLink size={11}/>
                                    </Link>
                                </p>
                            )}
                        </>
                    )}
                </FormField>
            )}
        </div>
    );
};
