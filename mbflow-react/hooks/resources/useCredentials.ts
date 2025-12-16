/**
 * Hook for managing credentials data
 * Single Responsibility: Handles fetching and managing credentials
 */

import { useState, useEffect, useCallback } from 'react';
import {
  credentialsApi,
  Credential,
  CredentialType,
  CredentialWithSecrets,
  CreateAPIKeyRequest,
  CreateBasicAuthRequest,
  CreateOAuth2Request,
  CreateServiceAccountRequest,
  CreateCustomCredentialRequest,
} from '@/services/credentialsService';
import { toast } from '@/lib/toast';

export interface CredentialsState {
  credentials: Credential[];
  loading: boolean;
  error: string | null;
}

export interface CredentialsActions {
  loadCredentials: (provider?: string) => Promise<void>;
  createCredential: (type: CredentialType, data: any) => Promise<boolean>;
  updateCredential: (id: string, name: string, description?: string) => Promise<boolean>;
  deleteCredential: (id: string) => Promise<boolean>;
  getSecrets: (id: string) => Promise<CredentialWithSecrets | null>;
}

export const useCredentials = (): CredentialsState & CredentialsActions => {
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadCredentials = useCallback(async (provider?: string) => {
    setLoading(true);
    setError(null);
    try {
      const response = await credentialsApi.listCredentials(provider);
      setCredentials(response.data.credentials || []);
    } catch (err: any) {
      console.error('Failed to load credentials:', err);
      const errorMsg = err.response?.data?.error || 'Failed to load credentials';
      setError(errorMsg);
      if (err.response?.status !== 404) {
        toast.error('Load Failed', errorMsg);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCredentials();
  }, [loadCredentials]);

  const createCredential = useCallback(async (type: CredentialType, data: any): Promise<boolean> => {
    try {
      switch (type) {
        case 'api_key':
          await credentialsApi.createAPIKey(data as CreateAPIKeyRequest);
          break;
        case 'basic_auth':
          await credentialsApi.createBasicAuth(data as CreateBasicAuthRequest);
          break;
        case 'oauth2':
          await credentialsApi.createOAuth2(data as CreateOAuth2Request);
          break;
        case 'service_account':
          await credentialsApi.createServiceAccount(data as CreateServiceAccountRequest);
          break;
        case 'custom':
          await credentialsApi.createCustom(data as CreateCustomCredentialRequest);
          break;
        default:
          throw new Error(`Unknown credential type: ${type}`);
      }
      toast.success('Success', 'Credential created successfully.');
      await loadCredentials();
      return true;
    } catch (err: any) {
      console.error('Failed to create credential:', err);
      toast.error('Create Failed', err.response?.data?.error || 'Failed to create credential.');
      return false;
    }
  }, [loadCredentials]);

  const updateCredential = useCallback(async (
    id: string,
    name: string,
    description?: string
  ): Promise<boolean> => {
    try {
      await credentialsApi.updateCredential(id, { name, description });
      toast.success('Success', 'Credential updated successfully.');
      await loadCredentials();
      return true;
    } catch (err: any) {
      console.error('Failed to update credential:', err);
      toast.error('Update Failed', err.response?.data?.error || 'Failed to update credential.');
      return false;
    }
  }, [loadCredentials]);

  const deleteCredential = useCallback(async (id: string): Promise<boolean> => {
    try {
      await credentialsApi.deleteCredential(id);
      toast.success('Success', 'Credential deleted successfully.');
      await loadCredentials();
      return true;
    } catch (err: any) {
      console.error('Failed to delete credential:', err);
      toast.error('Delete Failed', err.response?.data?.error || 'Failed to delete credential.');
      return false;
    }
  }, [loadCredentials]);

  const getSecrets = useCallback(async (id: string): Promise<CredentialWithSecrets | null> => {
    try {
      const response = await credentialsApi.getCredentialSecrets(id);
      return response.data;
    } catch (err: any) {
      console.error('Failed to get credential secrets:', err);
      if (err.response?.status === 410) {
        toast.error('Expired', 'This credential has expired.');
      } else {
        toast.error('Failed', err.response?.data?.error || 'Failed to get secrets.');
      }
      return null;
    }
  }, []);

  return {
    credentials,
    loading,
    error,
    loadCredentials,
    createCredential,
    updateCredential,
    deleteCredential,
    getSecrets,
  };
};
