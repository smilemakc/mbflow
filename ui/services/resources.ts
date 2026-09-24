import { apiClient } from '../lib/api';

/** Extract a filename from a Content-Disposition header, if present. */
const filenameFromDisposition = (header: unknown): string | undefined => {
  if (typeof header !== 'string') return undefined;
  const star = /filename\*=(?:UTF-8'')?([^;]+)/i.exec(header);
  if (star) {
    try {
      return decodeURIComponent(star[1].trim().replace(/^"|"$/g, ''));
    } catch {
      /* fall through */
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(header);
  return plain ? plain[1].trim() : undefined;
};

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
  downloadFile: async (resourceId: string, fileId: string, fileName?: string) => {
    const response = await apiClient.get<Blob>(
      `/resources/${resourceId}/files/${fileId}/download`,
      { responseType: 'blob', timeout: 0 },
    );
    const objectUrl = URL.createObjectURL(response.data);
    const link = document.createElement('a');
    link.href = objectUrl;
    link.download = fileName || filenameFromDisposition(response.headers['content-disposition']) || fileId;
    document.body.appendChild(link);
    try {
      link.click();
    } finally {
      document.body.removeChild(link);
      // Defer revocation so the browser has a chance to start the download
      setTimeout(() => URL.revokeObjectURL(objectUrl), 0);
    }
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
