/**
 * Test fixtures for components/resources, components/credentials and components/modals.
 * Shapes mirror the Go handlers (handlers_resources.go, handlers_accounts.go,
 * handlers_credentials.go, handlers_rental_keys.go, handlers_file_storage.go).
 */
import type { Account, FileMetadata, FileStorageResource, Transaction } from '@/services/resources';
import type { Credential } from '@/services/credentialsService';
import type { MultimodalUsage, RentalKey, UsageRecord } from '@/services/rentalKeyService';

export const storage = (overrides: Partial<FileStorageResource> = {}): FileStorageResource => ({
  id: 'res-1',
  type: 'file_storage',
  name: 'Primary storage',
  description: 'Main bucket',
  status: 'active',
  storage_limit_bytes: 1024 * 1024 * 1024,
  used_storage_bytes: 512 * 1024 * 1024,
  file_count: 3,
  usage_percent: 50,
  created_at: '2026-03-15T10:00:00Z',
  updated_at: '2026-03-15T10:00:00Z',
  ...overrides,
});

export const account = (overrides: Partial<Account> = {}): Account => ({
  id: 'acc-1',
  user_id: 'user-1',
  balance: 125.5,
  currency: 'USD',
  status: 'active',
  ...overrides,
});

export const transaction = (overrides: Partial<Transaction> = {}): Transaction => ({
  id: 'tx-1',
  type: 'deposit',
  amount: 10,
  currency: 'USD',
  status: 'completed',
  description: 'Top up',
  created_at: '2026-03-15T10:00:00Z',
  ...overrides,
});

export const fileMeta = (overrides: Partial<FileMetadata> = {}): FileMetadata => ({
  id: 'file-1',
  name: 'report.pdf',
  size: 2048,
  mime_type: 'application/pdf',
  checksum: 'abc',
  created_at: '2026-03-15T10:00:00Z',
  updated_at: '2026-03-15T10:00:00Z',
  ...overrides,
});

export const credential = (overrides: Partial<Credential> = {}): Credential => ({
  id: 'cred-1',
  name: 'OpenAI key',
  description: 'Production key',
  status: 'active',
  credential_type: 'api_key',
  provider: 'openai',
  usage_count: 7,
  created_at: '2026-03-15T10:00:00Z',
  updated_at: '2026-03-15T10:00:00Z',
  fields: ['api_key'],
  ...overrides,
});

export const usage = (overrides: Partial<MultimodalUsage> = {}): MultimodalUsage => ({
  prompt_tokens: 1000,
  completion_tokens: 500,
  image_input_tokens: 2000,
  image_output_tokens: 0,
  audio_input_tokens: 30,
  audio_output_tokens: 0,
  video_input_tokens: 0,
  video_output_tokens: 4_000_000,
  total: 4_003_530,
  ...overrides,
});

export const rentalKey = (overrides: Partial<RentalKey> = {}): RentalKey => ({
  id: 'rk-1',
  name: 'Team GPT key',
  description: 'Shared key',
  status: 'active',
  provider: 'openai',
  created_at: '2026-03-15T10:00:00Z',
  updated_at: '2026-03-15T10:00:00Z',
  requests_today: 10,
  tokens_this_month: 1500,
  total_requests: 2500,
  total_usage: usage(),
  total_cost: 12.345,
  ...overrides,
});

export const usageRecord = (overrides: Partial<UsageRecord> = {}): UsageRecord => ({
  id: 'u-1',
  model: 'gpt-4o',
  usage: usage({ total: 1500 }),
  estimated_cost: 0.01234,
  status: 'success',
  response_time_ms: 340,
  created_at: '2026-03-15T10:00:00Z',
  ...overrides,
});
