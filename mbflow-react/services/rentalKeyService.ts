import { apiClient } from '../lib/api';

// LLM Provider types
export type LLMProviderType = 'openai' | 'anthropic' | 'google_ai';

// Provisioner types
export type ProvisionerType = 'manual' | 'auto_openai' | 'auto_anthropic' | 'auto_google';

// Resource status
export type ResourceStatus = 'active' | 'suspended' | 'deleted';

// Multimodal usage structure
export interface MultimodalUsage {
  prompt_tokens: number;
  completion_tokens: number;
  image_input_tokens: number;
  image_output_tokens: number;
  audio_input_tokens: number;
  audio_output_tokens: number;
  video_input_tokens: number;
  video_output_tokens: number;
  total: number;
}

// User-facing rental key (without API key value)
export interface RentalKey {
  id: string;
  name: string;
  description?: string;
  status: ResourceStatus;
  provider: LLMProviderType;
  created_at: string;
  updated_at: string;
  last_used_at?: string;
  metadata?: Record<string, unknown>;

  // Usage limits
  daily_request_limit?: number;
  monthly_token_limit?: number;

  // Current usage
  requests_today: number;
  tokens_this_month: number;

  // Aggregated statistics
  total_requests: number;
  total_usage: MultimodalUsage;
  total_cost: number;
}

// Admin-facing rental key (includes additional fields)
export interface AdminRentalKey extends RentalKey {
  owner_id: string;
  pricing_plan_id?: string;
  created_by?: string;
  provisioner_type: ProvisionerType;
}

// Usage record
export interface UsageRecord {
  id: string;
  model: string;
  usage: MultimodalUsage;
  estimated_cost: number;
  execution_id?: string;
  workflow_id?: string;
  node_id?: string;
  status: string;
  error_message?: string;
  response_time_ms?: number;
  created_at: string;
}

// Usage summary
export interface UsageSummary {
  total_requests: number;
  total_cost: number;
  total_usage: MultimodalUsage;
}

// Admin request types
export interface CreateRentalKeyRequest {
  owner_id: string;
  name: string;
  description?: string;
  provider: LLMProviderType;
  api_key: string;
  provider_config?: Record<string, unknown>;
  daily_request_limit?: number;
  monthly_token_limit?: number;
  pricing_plan_id?: string;
  provisioner_type?: ProvisionerType;
}

export interface UpdateRentalKeyRequest {
  name?: string;
  description?: string;
  status?: ResourceStatus;
  daily_request_limit?: number;
  monthly_token_limit?: number;
  provider_config?: Record<string, unknown>;
}

export interface RotateKeyRequest {
  new_api_key: string;
}

// Admin filter options
export interface RentalKeyFilter {
  provider?: LLMProviderType;
  status?: ResourceStatus;
  owner_id?: string;
  created_by?: string;
  limit?: number;
  offset?: number;
}

// ============================================================================
// User API - for regular users to view their own rental keys
// ============================================================================
export const rentalKeyApi = {
  // List user's rental keys
  listRentalKeys: (provider?: LLMProviderType) => {
    const params = provider ? `?provider=${encodeURIComponent(provider)}` : '';
    return apiClient.get<{ rental_keys: RentalKey[] }>(`/rental-keys${params}`);
  },

  // Get a specific rental key (without API key value)
  getRentalKey: (id: string) =>
    apiClient.get<RentalKey>(`/rental-keys/${id}`),

  // Get usage history for a rental key
  getUsageHistory: (id: string, limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit !== undefined) params.append('limit', limit.toString());
    if (offset !== undefined) params.append('offset', offset.toString());
    const queryString = params.toString();
    return apiClient.get<{ usage: UsageRecord[]; limit: number; offset: number }>(
      `/rental-keys/${id}/usage${queryString ? '?' + queryString : ''}`
    );
  },

  // Get usage summary for a rental key
  getUsageSummary: (id: string) =>
    apiClient.get<{ summary: UsageSummary }>(`/rental-keys/${id}/summary`),
};

// ============================================================================
// Admin API - for administrators to manage rental keys
// ============================================================================
export const rentalKeyAdminApi = {
  // Create a new rental key
  createRentalKey: (request: CreateRentalKeyRequest) =>
    apiClient.post<AdminRentalKey>('/admin/rental-keys', request),

  // List all rental keys with optional filtering
  listAllRentalKeys: (filter?: RentalKeyFilter) => {
    const params = new URLSearchParams();
    if (filter?.provider) params.append('provider', filter.provider);
    if (filter?.status) params.append('status', filter.status);
    if (filter?.owner_id) params.append('owner_id', filter.owner_id);
    if (filter?.created_by) params.append('created_by', filter.created_by);
    if (filter?.limit !== undefined) params.append('limit', filter.limit.toString());
    if (filter?.offset !== undefined) params.append('offset', filter.offset.toString());
    const queryString = params.toString();
    return apiClient.get<{ rental_keys: AdminRentalKey[]; total: number; limit: number; offset: number }>(
      `/admin/rental-keys${queryString ? '?' + queryString : ''}`
    );
  },

  // Get a specific rental key
  getRentalKey: (id: string) =>
    apiClient.get<AdminRentalKey>(`/admin/rental-keys/${id}`),

  // Update a rental key
  updateRentalKey: (id: string, request: UpdateRentalKeyRequest) =>
    apiClient.put<AdminRentalKey>(`/admin/rental-keys/${id}`, request),

  // Rotate API key
  rotateAPIKey: (id: string, request: RotateKeyRequest) =>
    apiClient.post<{ message: string }>(`/admin/rental-keys/${id}/rotate-key`, request),

  // Delete a rental key
  deleteRentalKey: (id: string) =>
    apiClient.delete<{ message: string }>(`/admin/rental-keys/${id}`),

  // Reset daily usage counters (typically called by scheduled job)
  resetDailyUsage: () =>
    apiClient.post<{ message: string }>('/admin/rental-keys/reset-daily'),

  // Reset monthly usage counters (typically called by scheduled job)
  resetMonthlyUsage: () =>
    apiClient.post<{ message: string }>('/admin/rental-keys/reset-monthly'),
};

// ============================================================================
// Helper functions
// ============================================================================

// Get human-readable provider name
export const getProviderLabel = (provider: LLMProviderType): string => {
  const labels: Record<LLMProviderType, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    google_ai: 'Google AI',
  };
  return labels[provider] || provider;
};

// Get provisioner type label
export const getProvisionerLabel = (provisioner: ProvisionerType): string => {
  const labels: Record<ProvisionerType, string> = {
    manual: 'Manual',
    auto_openai: 'Auto (OpenAI)',
    auto_anthropic: 'Auto (Anthropic)',
    auto_google: 'Auto (Google)',
  };
  return labels[provisioner] || provisioner;
};

// Get status badge color
export const getStatusColor = (status: ResourceStatus): string => {
  const colors: Record<ResourceStatus, string> = {
    active: 'green',
    suspended: 'yellow',
    deleted: 'red',
  };
  return colors[status] || 'gray';
};

// Format token count for display
export const formatTokenCount = (tokens: number): string => {
  if (tokens >= 1_000_000_000) {
    return `${(tokens / 1_000_000_000).toFixed(2)}B`;
  }
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(2)}M`;
  }
  if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(2)}K`;
  }
  return tokens.toString();
};

// Calculate usage percentage for limits
export const calculateUsagePercent = (current: number, limit?: number): number | null => {
  if (limit === undefined || limit === null || limit === 0) {
    return null;
  }
  return Math.min(100, (current / limit) * 100);
};

// Check if rental key is near limit
export const isNearLimit = (current: number, limit?: number, threshold = 80): boolean => {
  const percent = calculateUsagePercent(current, limit);
  return percent !== null && percent >= threshold;
};

// Available LLM providers for selection
export const LLM_PROVIDERS: { value: LLMProviderType; label: string }[] = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'google_ai', label: 'Google AI' },
];

// Provisioner types for admin selection
export const PROVISIONER_TYPES: { value: ProvisionerType; label: string }[] = [
  { value: 'manual', label: 'Manual' },
  { value: 'auto_openai', label: 'Auto (OpenAI)' },
  { value: 'auto_anthropic', label: 'Auto (Anthropic)' },
  { value: 'auto_google', label: 'Auto (Google)' },
];
