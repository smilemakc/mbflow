/**
 * Hook for managing rental keys data
 * Single Responsibility: Handles fetching and managing user's rental keys
 */

import { useState, useEffect, useCallback } from 'react';
import {
  rentalKeyApi,
  RentalKey,
  LLMProviderType,
  UsageRecord,
  UsageSummary,
} from '@/services/rentalKeyService';
import { toast } from '@/lib/toast';
import { getErrorMessage } from '@/lib/api';

export interface RentalKeysState {
  rentalKeys: RentalKey[];
  loading: boolean;
  error: string | null;
}

export interface RentalKeysActions {
  loadRentalKeys: (provider?: LLMProviderType) => Promise<void>;
  getRentalKey: (id: string) => Promise<RentalKey | null>;
  getUsageHistory: (id: string, limit?: number, offset?: number) => Promise<UsageRecord[]>;
  getUsageSummary: (id: string) => Promise<UsageSummary | null>;
  refresh: () => Promise<void>;
}

export const useRentalKeys = (initialProvider?: LLMProviderType): RentalKeysState & RentalKeysActions => {
  const [rentalKeys, setRentalKeys] = useState<RentalKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [providerFilter, setProviderFilter] = useState<LLMProviderType | undefined>(initialProvider);

  const loadRentalKeys = useCallback(async (provider?: LLMProviderType) => {
    setLoading(true);
    setError(null);
    if (provider !== undefined) {
      setProviderFilter(provider);
    }
    try {
      const response = await rentalKeyApi.listRentalKeys(provider ?? providerFilter);
      setRentalKeys(response.data.rental_keys || []);
    } catch (error: unknown) {
      console.error('Failed to load rental keys:', error);
      const errorMsg = getErrorMessage(error);
      setError(errorMsg);
      toast.error('Load Failed', errorMsg);
    } finally {
      setLoading(false);
    }
  }, [providerFilter]);

  useEffect(() => {
    loadRentalKeys();
  }, []);

  const getRentalKey = useCallback(async (id: string): Promise<RentalKey | null> => {
    try {
      const response = await rentalKeyApi.getRentalKey(id);
      return response.data;
    } catch (error: unknown) {
      console.error('Failed to get rental key:', error);
      toast.error('Failed', getErrorMessage(error));
      return null;
    }
  }, []);

  const getUsageHistory = useCallback(async (
    id: string,
    limit?: number,
    offset?: number
  ): Promise<UsageRecord[]> => {
    try {
      const response = await rentalKeyApi.getUsageHistory(id, limit, offset);
      return response.data.usage || [];
    } catch (error: unknown) {
      console.error('Failed to get usage history:', error);
      toast.error('Failed', getErrorMessage(error));
      return [];
    }
  }, []);

  const getUsageSummary = useCallback(async (id: string): Promise<UsageSummary | null> => {
    try {
      const response = await rentalKeyApi.getUsageSummary(id);
      return response.data.summary;
    } catch (error: unknown) {
      console.error('Failed to get usage summary:', error);
      toast.error('Failed', getErrorMessage(error));
      return null;
    }
  }, []);

  const refresh = useCallback(async () => {
    await loadRentalKeys(providerFilter);
  }, [loadRentalKeys, providerFilter]);

  return {
    rentalKeys,
    loading,
    error,
    loadRentalKeys,
    getRentalKey,
    getUsageHistory,
    getUsageSummary,
    refresh,
  };
};
