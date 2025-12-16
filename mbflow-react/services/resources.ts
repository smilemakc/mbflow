import { apiClient } from '../lib/api';

export interface FileStorageResource {
  id: string;
  type: string;
  name: string;
  description: string;
  status: string;
  storage_limit_bytes: number;
  used_storage_bytes: number;
  file_count: number;
  usage_percent: number;
  created_at: string;
  updated_at: string;
}

export interface Account {
  id: string;
  user_id: string;
  balance: number;
  currency: string;
  status: string;
}

export interface Transaction {
  id: string;
  type: string;
  amount: number;
  currency: string;
  status: string;
  description: string;
  created_at: string;
}

export interface PricingPlan {
  id: string;
  name: string;
  description: string;
  price_per_unit: number;
  storage_limit_bytes: number | null;
  is_free: boolean;
}

export interface FileMetadata {
  id: string;
  name: string;
  size: number;
  mime_type: string;
  checksum: string;
  created_at: string;
  updated_at: string;
  expires_at?: string;
}

export const resourcesApi = {
  listResources: () => apiClient.get<{ resources: FileStorageResource[] }>('/resources'),
  getResource: (id: string) => apiClient.get<FileStorageResource>(`/resources/${id}`),
  createFileStorage: (name: string, description?: string) =>
    apiClient.post<FileStorageResource>('/resources/file-storage', { name, description }),
  deleteResource: (id: string) => apiClient.delete(`/resources/${id}`),
  updateResource: (id: string, name: string, description?: string) =>
    apiClient.put<FileStorageResource>(`/resources/${id}`, { name, description }),

  listPricingPlans: () => apiClient.get<{ plans: PricingPlan[] }>('/resources/pricing-plans'),

  // File operations
  uploadFile: (resourceId: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.post<FileMetadata>(`/resources/${resourceId}/files`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
  listFiles: (resourceId: string, limit?: number, offset?: number) =>
    apiClient.get<{ files: FileMetadata[], total: number, limit: number, offset: number }>(
      `/resources/${resourceId}/files?limit=${limit || 50}&offset=${offset || 0}`
    ),
  getFileMetadata: (resourceId: string, fileId: string) =>
    apiClient.get<FileMetadata>(`/resources/${resourceId}/files/${fileId}`),
  downloadFile: (resourceId: string, fileId: string) => {
    const token = localStorage.getItem('token');
    const url = `${apiClient.defaults.baseURL}/resources/${resourceId}/files/${fileId}/download`;
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', '');
    if (token) {
      link.setAttribute('Authorization', `Bearer ${token}`);
    }
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  },
  deleteFile: (resourceId: string, fileId: string) =>
    apiClient.delete(`/resources/${resourceId}/files/${fileId}`),

  getAccount: () => apiClient.get<Account>('/account'),
  deposit: (amount: number, idempotencyKey: string) =>
    apiClient.post<Transaction>('/account/deposit', { amount, idempotency_key: idempotencyKey }),
  listTransactions: (limit?: number, offset?: number) =>
    apiClient.get<{ transactions: Transaction[], total: number }>(
      `/account/transactions?limit=${limit || 20}&offset=${offset || 0}`
    ),
};
