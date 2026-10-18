import { apiClient, ApiListResponse } from '@/lib/api';

/** JSON Schema as stored by the backend; "$ref" points to ApiClient.schemas. */
export type JsonSchema = Record<string, any>;

export type ApiAuthType = 'none' | 'api_key' | 'bearer' | 'basic' | 'oauth2_client_credentials';
export type ApiKeyLocation = 'header' | 'query' | 'cookie';
export type ParameterLocation = 'path' | 'query' | 'header' | 'cookie';

export interface ApiClientAuth {
  type: ApiAuthType;
  credential_id?: string;
  in?: ApiKeyLocation;
  name?: string;
  scheme_name?: string;
  token_url?: string;
  scopes?: string[];
}

export interface ApiServer {
  url: string;
  description?: string;
}

export interface ApiSecurityScheme {
  name: string;
  type: string;
  scheme?: string;
  bearer_format?: string;
  in?: string;
  param_name?: string;
  description?: string;
  token_url?: string;
  scopes?: string[];
}

export interface ApiParameter {
  name: string;
  in: ParameterLocation;
  description?: string;
  required?: boolean;
  deprecated?: boolean;
  style?: string;
  explode?: boolean;
  schema?: JsonSchema;
  example?: unknown;
  content_type?: string;
}

export interface ApiMediaType {
  content_type: string;
  schema?: JsonSchema;
  example?: unknown;
}

export interface ApiRequestBody {
  description?: string;
  required?: boolean;
  content: ApiMediaType[];
}

export interface ApiResponse {
  status: string;
  description?: string;
  content?: ApiMediaType[];
}

export interface ApiOperation {
  id: string;
  operation_id?: string;
  method: string;
  path: string;
  summary?: string;
  description?: string;
  tags?: string[];
  deprecated?: boolean;
  parameters?: ApiParameter[];
  request_body?: ApiRequestBody;
  responses?: ApiResponse[];
  servers?: ApiServer[];
  security?: string[][];
  no_auth?: boolean;
}

export interface ApiSpecWarning {
  code: string;
  message: string;
}

export interface ApiClientSummary {
  id: string;
  name: string;
  description?: string;
  title: string;
  api_version?: string;
  spec_version: string;
  base_url?: string;
  auth: ApiClientAuth;
  operation_count: number;
  warnings: ApiSpecWarning[];
  created_at: string;
  updated_at: string;
}

export interface ApiClient extends ApiClientSummary {
  owner_id: string;
  servers: ApiServer[];
  security_schemes: ApiSecurityScheme[];
  default_headers: Record<string, string>;
  operations: ApiOperation[];
  schemas: Record<string, JsonSchema>;
  spec_size: number;
}

export interface ApiOperationSummary {
  id: string;
  method: string;
  path: string;
  summary?: string;
  tags?: string[];
  deprecated?: boolean;
}

export interface ApiSpecPreview {
  title: string;
  description?: string;
  api_version?: string;
  spec_version: string;
  base_url?: string;
  servers: ApiServer[];
  security_schemes: ApiSecurityScheme[];
  suggested_auth: ApiClientAuth;
  operation_count: number;
  operations: ApiOperationSummary[];
  warnings: ApiSpecWarning[];
}

export interface ApiClientSettings {
  name?: string;
  description?: string;
  base_url?: string;
  auth?: ApiClientAuth;
  default_headers?: Record<string, string>;
}

export interface ApiSpecChanges {
  added: string[];
  removed: string[];
}

/** Parsing a large specification takes longer than the default request timeout. */
const SPEC_UPLOAD_TIMEOUT = 120_000;

const specForm = (file: File, settings: ApiClientSettings = {}): FormData => {
  const form = new FormData();
  form.append('file', file);
  if (settings.name) form.append('name', settings.name);
  if (settings.description) form.append('description', settings.description);
  if (settings.base_url) form.append('base_url', settings.base_url);
  if (settings.auth) form.append('auth', JSON.stringify(settings.auth));
  if (settings.default_headers) form.append('default_headers', JSON.stringify(settings.default_headers));
  return form;
};

const multipart = { headers: { 'Content-Type': 'multipart/form-data' }, timeout: SPEC_UPLOAD_TIMEOUT };

export const apiClientService = {
  preview: (file: File) =>
    apiClient.post<ApiSpecPreview>('/api-clients/preview', specForm(file), multipart),

  create: (file: File, settings: ApiClientSettings) =>
    apiClient.post<ApiClient>('/api-clients', specForm(file, settings), multipart),

  list: () =>
    apiClient.get<ApiListResponse<ApiClientSummary>>('/api-clients', { params: { limit: 100 } }),

  get: (id: string) =>
    apiClient.get<ApiClient>(`/api-clients/${id}`, { timeout: SPEC_UPLOAD_TIMEOUT }),

  update: (id: string, settings: ApiClientSettings) =>
    apiClient.put<ApiClient>(`/api-clients/${id}`, settings),

  replaceSpec: (id: string, file: File) =>
    apiClient.put<{ client: ApiClient; changes: ApiSpecChanges }>(`/api-clients/${id}/spec`, specForm(file), multipart),

  downloadSpec: (id: string) =>
    apiClient.get<Blob>(`/api-clients/${id}/spec`, { responseType: 'blob', timeout: SPEC_UPLOAD_TIMEOUT }),

  remove: (id: string) =>
    apiClient.delete(`/api-clients/${id}`),
};
