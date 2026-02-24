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
import { getErrorMessage } from '@/lib/api';

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
    } catch (error: unknown) {
      console.error('Failed to load credentials:', error);
      const errorMsg = getErrorMessage(error);
      setError(errorMsg);
      toast.error('Load Failed', errorMsg);
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
    } catch (error: unknown) {
      console.error('Failed to create credential:', error);
      toast.error('Create Failed', getErrorMessage(error));
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
    } catch (error: unknown) {
      console.error('Failed to update credential:', error);
      toast.error('Update Failed', getErrorMessage(error));
      return false;
    }
  }, [loadCredentials]);

  const deleteCredential = useCallback(async (id: string): Promise<boolean> => {
    try {
      await credentialsApi.deleteCredential(id);
      toast.success('Success', 'Credential deleted successfully.');
      await loadCredentials();
      return true;
    } catch (error: unknown) {
      console.error('Failed to delete credential:', error);
      toast.error('Delete Failed', getErrorMessage(error));
      return false;
    }
  }, [loadCredentials]);

  const getSecrets = useCallback(async (id: string): Promise<CredentialWithSecrets | null> => {
    try {
      const response = await credentialsApi.getCredentialSecrets(id);
      return response.data;
    } catch (error: unknown) {
      console.error('Failed to get credential secrets:', error);
      toast.error('Failed', getErrorMessage(error));
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
