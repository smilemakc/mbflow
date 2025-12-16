import { apiClient } from '../lib/api';

// Credential types
export type CredentialType = 'api_key' | 'basic_auth' | 'oauth2' | 'service_account' | 'custom';

// Response types
export interface Credential {
  id: string;
  name: string;
  description: string;
  status: string;
  credential_type: CredentialType;
  provider?: string;
  expires_at?: string;
  last_used_at?: string;
  usage_count: number;
  created_at: string;
  updated_at: string;
  fields: string[];
}

export interface CredentialWithSecrets extends Credential {
  data: Record<string, string>;
}

// Request types
export interface CreateAPIKeyRequest {
  name: string;
  description?: string;
  provider?: string;
  api_key: string;
}

export interface CreateBasicAuthRequest {
  name: string;
  description?: string;
  provider?: string;
  username: string;
  password: string;
}

export interface CreateOAuth2Request {
  name: string;
  description?: string;
  provider?: string;
  client_id: string;
  client_secret: string;
  access_token?: string;
  refresh_token?: string;
  token_url?: string;
  scopes?: string;
}

export interface CreateServiceAccountRequest {
  name: string;
  description?: string;
  provider?: string;
  json_key: string;
}

export interface CreateCustomCredentialRequest {
  name: string;
  description?: string;
  provider?: string;
  data: Record<string, string>;
}

export interface UpdateCredentialRequest {
  name?: string;
  description?: string;
}

// API functions
export const credentialsApi = {
  // List all credentials (without secrets)
  listCredentials: (provider?: string) => {
    const params = provider ? `?provider=${encodeURIComponent(provider)}` : '';
    return apiClient.get<{ credentials: Credential[] }>(`/credentials${params}`);
  },

  // Get credential by ID (without secrets)
  getCredential: (id: string) =>
    apiClient.get<Credential>(`/credentials/${id}`),

  // Get credential with decrypted secrets (sensitive!)
  getCredentialSecrets: (id: string) =>
    apiClient.get<CredentialWithSecrets>(`/credentials/${id}/secrets`),

  // Create API key credential
  createAPIKey: (request: CreateAPIKeyRequest) =>
    apiClient.post<Credential>('/credentials/api-key', request),

  // Create basic auth credential
  createBasicAuth: (request: CreateBasicAuthRequest) =>
    apiClient.post<Credential>('/credentials/basic-auth', request),

  // Create OAuth2 credential
  createOAuth2: (request: CreateOAuth2Request) =>
    apiClient.post<Credential>('/credentials/oauth2', request),

  // Create service account credential
  createServiceAccount: (request: CreateServiceAccountRequest) =>
    apiClient.post<Credential>('/credentials/service-account', request),

  // Create custom credential
  createCustom: (request: CreateCustomCredentialRequest) =>
    apiClient.post<Credential>('/credentials/custom', request),

  // Update credential metadata
  updateCredential: (id: string, request: UpdateCredentialRequest) =>
    apiClient.put<Credential>(`/credentials/${id}`, request),

  // Delete credential
  deleteCredential: (id: string) =>
    apiClient.delete(`/credentials/${id}`),
};

// Helper functions
export const getCredentialTypeLabel = (type: CredentialType): string => {
  const labels: Record<CredentialType, string> = {
    api_key: 'API Key',
    basic_auth: 'Basic Auth',
    oauth2: 'OAuth2',
    service_account: 'Service Account',
    custom: 'Custom',
  };
  return labels[type] || type;
};


// Common providers
export const COMMON_PROVIDERS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'google', label: 'Google Cloud' },
  { value: 'aws', label: 'AWS' },
  { value: 'azure', label: 'Azure' },
  { value: 'github', label: 'GitHub' },
  { value: 'gitlab', label: 'GitLab' },
  { value: 'slack', label: 'Slack' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'sendgrid', label: 'SendGrid' },
  { value: 'stripe', label: 'Stripe' },
  { value: 'twilio', label: 'Twilio' },
  { value: 'other', label: 'Other' },
];
