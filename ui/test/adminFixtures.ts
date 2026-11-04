/**
 * Fixtures for admin (users, rental keys) and API client tests.
 * Shapes follow the Go REST contract (handlers_auth.go, handlers_rental_keys_admin.go,
 * handlers_api_clients.go).
 */
import type { User } from '@/types/auth';
import type { AdminRentalKey } from '@/services/rentalKeyService';
import type { ApiClient, ApiOperation, ApiSpecPreview } from '@/services/apiClientService';
import type { Credential } from '@/services/credentialsService';

export const adminUser = (id: string, overrides: Partial<User> = {}): User => ({
  id,
  email: `${id}@example.com`,
  username: id,
  is_active: true,
  is_admin: false,
  roles: ['user'],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

export const ADMIN_USERS: User[] = [
  adminUser('alice', { full_name: 'Alice Liddell', is_admin: true }),
  adminUser('bob', { full_name: 'Bob' }),
  adminUser('carol'),
];

const usage = {
  prompt_tokens: 0,
  completion_tokens: 0,
  image_input_tokens: 0,
  image_output_tokens: 0,
  audio_input_tokens: 0,
  audio_output_tokens: 0,
  video_input_tokens: 0,
  video_output_tokens: 0,
  total: 0,
};

export const adminRentalKey = (id: string, overrides: Partial<AdminRentalKey> = {}): AdminRentalKey => ({
  id,
  name: `Key ${id}`,
  status: 'active',
  provider: 'openai',
  created_at: '2024-03-05T10:00:00Z',
  updated_at: '2024-03-05T10:00:00Z',
  requests_today: 0,
  tokens_this_month: 0,
  total_requests: 0,
  total_usage: usage,
  total_cost: 0,
  owner_id: 'alice',
  provisioner_type: 'manual',
  ...overrides,
});

export const credential = (id: string, credential_type: Credential['credential_type'], name = id): Credential => ({
  id,
  name,
  description: '',
  status: 'active',
  credential_type,
  usage_count: 0,
  created_at: '',
  updated_at: '',
  fields: [],
});

export const operation = (id: string, overrides: Partial<ApiOperation> = {}): ApiOperation => ({
  id,
  method: 'GET',
  path: `/${id}`,
  ...overrides,
});

export const apiClientFixture = (id: string, overrides: Partial<ApiClient> = {}): ApiClient => ({
  id,
  owner_id: 'alice',
  name: `Client ${id}`,
  title: 'Petstore',
  spec_version: '3.0.3',
  auth: { type: 'none' },
  operation_count: 0,
  warnings: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  servers: [],
  security_schemes: [],
  default_headers: {},
  operations: [],
  schemas: {},
  spec_size: 100,
  ...overrides,
});

export const specPreview = (overrides: Partial<ApiSpecPreview> = {}): ApiSpecPreview => ({
  title: 'Petstore',
  spec_version: '3.0.3',
  servers: [],
  security_schemes: [],
  suggested_auth: { type: 'none' },
  operation_count: 3,
  operations: [],
  warnings: [],
  ...overrides,
});
