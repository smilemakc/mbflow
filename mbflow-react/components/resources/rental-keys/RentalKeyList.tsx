/**
 * RentalKeyList component
 * Displays a list of rental keys with filtering options.
 */

import React, { useState, useEffect } from 'react';
import { Key, AlertCircle, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui';
import { RentalKeyCard } from './RentalKeyCard';
import { RentalKeyUsageModal } from './RentalKeyUsageModal';
import {
  RentalKey,
  LLMProviderType,
  rentalKeyApi,
  LLM_PROVIDERS,
} from '@/services/rentalKeyService';
import { useTranslation } from '@/store/translations';
import { getErrorMessage } from '@/lib/api';

interface RentalKeyListProps {
  className?: string;
}

export const RentalKeyList: React.FC<RentalKeyListProps> = ({ className }) => {
  const t = useTranslation();
  const [rentalKeys, setRentalKeys] = useState<RentalKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [providerFilter, setProviderFilter] = useState<LLMProviderType | ''>('');
  const [selectedKey, setSelectedKey] = useState<RentalKey | null>(null);
  const [usageModalOpen, setUsageModalOpen] = useState(false);

  const fetchRentalKeys = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await rentalKeyApi.listRentalKeys(
        providerFilter || undefined
      );
      setRentalKeys(response.data.rental_keys || []);
    } catch (error: unknown) {
      setError(getErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRentalKeys();
  }, [providerFilter]);

  const handleViewUsage = (key: RentalKey) => {
    setSelectedKey(key);
    setUsageModalOpen(true);
  };

  const handleCloseUsageModal = () => {
    setUsageModalOpen(false);
    setSelectedKey(null);
  };

  if (loading && rentalKeys.length === 0) {
    return (
      <div className={`flex items-center justify-center py-12 ${className || ''}`}>
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className={className}>
      {/* Header with filters */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-4">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white flex items-center">
            <Key size={20} className="mr-2" />
            {t.rentalKeys?.title || 'Rental Keys'}
          </h2>
          <select
            value={providerFilter}
            onChange={(e) => setProviderFilter(e.target.value as LLMProviderType | '')}
            className="text-sm border border-slate-200 dark:border-slate-700 rounded-lg px-3 py-1.5 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
          >
            <option value="">{t.rentalKeys?.allProviders || 'All Providers'}</option>
            {LLM_PROVIDERS.map((provider) => (
              <option key={provider.value} value={provider.value}>
                {provider.label}
              </option>
            ))}
          </select>
        </div>
        <Button
          onClick={fetchRentalKeys}
          variant="ghost"
          size="sm"
          icon={<RefreshCw size={14} />}
          disabled={loading}
        >
          {t.common?.refresh || 'Refresh'}
        </Button>
      </div>

      {/* Error message */}
      {error && (
        <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-center text-red-700 dark:text-red-400">
          <AlertCircle size={16} className="mr-2" />
          {error}
        </div>
      )}

      {/* Empty state */}
      {!loading && rentalKeys.length === 0 && (
        <EmptyState providerFilter={providerFilter} />
      )}

      {/* Rental keys grid */}
      {rentalKeys.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {rentalKeys.map((key) => (
            <RentalKeyCard
              key={key.id}
              rentalKey={key}
              onViewUsage={handleViewUsage}
            />
          ))}
        </div>
      )}

      {/* Usage modal */}
      {selectedKey && (
        <RentalKeyUsageModal
          isOpen={usageModalOpen}
          onClose={handleCloseUsageModal}
          rentalKey={selectedKey}
        />
      )}
    </div>
  );
};

interface EmptyStateProps {
  providerFilter: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({ providerFilter }) => {
  const t = useTranslation();

  return (
    <div className="text-center py-12 px-4">
      <div className="inline-flex items-center justify-center w-16 h-16 bg-slate-100 dark:bg-slate-800 rounded-full mb-4">
        <Key size={32} className="text-slate-400" />
      </div>
      <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-2">
        {t.rentalKeys?.noKeysTitle || 'No Rental Keys'}
      </h3>
      <p className="text-slate-500 dark:text-slate-400 max-w-md mx-auto">
        {providerFilter
          ? t.rentalKeys?.noKeysForProvider || 'No rental keys found for this provider.'
          : t.rentalKeys?.noKeysDescription ||
            'You don\'t have any rental keys assigned yet. Contact your administrator to get access to LLM API keys.'}
      </p>
    </div>
  );
};

export default RentalKeyList;
