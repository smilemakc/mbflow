/**
 * Hook for managing resources data
 * Single Responsibility: Handles fetching and managing resources, account, and transactions
 */

import { useState, useEffect, useCallback } from 'react';
import { resourcesApi, FileStorageResource, Account, Transaction } from '@/services/resources.ts';
import { toast } from '@/lib/toast.ts';

export interface ResourcesState {
  resources: FileStorageResource[];
  account: Account | null;
  transactions: Transaction[];
  transactionsTotal: number;
  loading: boolean;
}

export interface ResourcesActions {
  loadData: () => Promise<void>;
  createStorage: (name: string, description: string) => Promise<boolean>;
  deleteResource: (id: string) => Promise<boolean>;
  deposit: (amount: number) => Promise<boolean>;
}

export const useResources = (): ResourcesState & ResourcesActions => {
  const [resources, setResources] = useState<FileStorageResource[]>([]);
  const [account, setAccount] = useState<Account | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [transactionsTotal, setTransactionsTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [resourcesRes, accountRes, transactionsRes] = await Promise.all([
        resourcesApi.listResources(),
        resourcesApi.getAccount(),
        resourcesApi.listTransactions(10, 0),
      ]);
      setResources(resourcesRes.data.resources || []);
      setAccount(accountRes.data);
      setTransactions(transactionsRes.data.transactions || []);
      setTransactionsTotal(transactionsRes.data.total || 0);
    } catch (error: any) {
      console.error('Failed to load data:', error);
      if (error.response?.status !== 404) {
        toast.error('Load Failed', 'Failed to load resources data. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const createStorage = useCallback(async (name: string, description: string): Promise<boolean> => {
    try {
      await resourcesApi.createFileStorage(name, description);
      toast.success('Success', 'File storage created successfully.');
      await loadData();
      return true;
    } catch (error: any) {
      console.error('Failed to create storage:', error);
      toast.error('Create Failed', error.response?.data?.message || 'Failed to create storage.');
      return false;
    }
  }, [loadData]);

  const deleteResource = useCallback(async (id: string): Promise<boolean> => {
    try {
      await resourcesApi.deleteResource(id);
      toast.success('Success', 'Resource deleted successfully.');
      await loadData();
      return true;
    } catch (error: any) {
      console.error('Failed to delete resource:', error);
      toast.error('Delete Failed', error.response?.data?.message || 'Failed to delete resource.');
      return false;
    }
  }, [loadData]);

  const deposit = useCallback(async (amount: number): Promise<boolean> => {
    try {
      const idempotencyKey = `deposit-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      await resourcesApi.deposit(amount, idempotencyKey);
      toast.success('Success', `Deposited ${amount.toFixed(2)} successfully.`);
      await loadData();
      return true;
    } catch (error: any) {
      console.error('Failed to deposit:', error);
      toast.error('Deposit Failed', error.response?.data?.message || 'Failed to process deposit.');
      return false;
    }
  }, [loadData]);

  return {
    resources,
    account,
    transactions,
    transactionsTotal,
    loading,
    loadData,
    createStorage,
    deleteResource,
    deposit,
  };
};
