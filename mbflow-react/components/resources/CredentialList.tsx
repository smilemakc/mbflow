/**
 * CredentialList component
 * Single Responsibility: Display and manage list of credentials
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Lock, Plus } from 'lucide-react';
import { Button } from '@/components/ui';
import { useTranslation } from '@/store/translations';
import { CredentialCard } from './CredentialCard';
import { CreateCredentialModal } from './CreateCredentialModal';
import { EditCredentialModal } from './EditCredentialModal';
import { ViewSecretsModal } from './ViewSecretsModal';
import {
  Credential,
  CredentialType,
  credentialsApi,
  getCredentialTypeLabel,
} from '@/services/credentialsService';

interface CredentialListProps {
  className?: string;
}

export const CredentialList: React.FC<CredentialListProps> = ({ className = '' }) => {
  const t = useTranslation();
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filter state
  const [filterType, setFilterType] = useState<CredentialType | ''>('');
  const [filterProvider, setFilterProvider] = useState('');

  // Modal states
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingCredential, setEditingCredential] = useState<Credential | null>(null);
  const [viewingCredential, setViewingCredential] = useState<Credential | null>(null);
  const [deletingCredential, setDeletingCredential] = useState<Credential | null>(null);

  const loadCredentials = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await credentialsApi.listCredentials(filterProvider || undefined);
      let filtered = response.data.credentials || [];

      // Apply type filter client-side
      if (filterType) {
        filtered = filtered.filter((c) => c.credential_type === filterType);
      }

      setCredentials(filtered);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load credentials');
    } finally {
      setLoading(false);
    }
  }, [filterType, filterProvider]);

  useEffect(() => {
    loadCredentials();
  }, [loadCredentials]);

  const handleCreate = async (type: CredentialType, data: any): Promise<boolean> => {
    try {
      switch (type) {
        case 'api_key':
          await credentialsApi.createAPIKey(data);
          break;
        case 'basic_auth':
          await credentialsApi.createBasicAuth(data);
          break;
        case 'oauth2':
          await credentialsApi.createOAuth2(data);
          break;
        case 'service_account':
          await credentialsApi.createServiceAccount(data);
          break;
        case 'custom':
          await credentialsApi.createCustom(data);
          break;
      }
      await loadCredentials();
      return true;
    } catch (err: any) {
      console.error('Failed to create credential:', err);
      return false;
    }
  };

  const handleDelete = async (credential: Credential) => {
    try {
      await credentialsApi.deleteCredential(credential.id);
      await loadCredentials();
      setDeletingCredential(null);
    } catch (err: any) {
      console.error('Failed to delete credential:', err);
    }
  };

  const handleEdit = async (id: string, name: string, description?: string): Promise<boolean> => {
    try {
      await credentialsApi.updateCredential(id, { name, description });
      await loadCredentials();
      return true;
    } catch (err: any) {
      console.error('Failed to update credential:', err);
      return false;
    }
  };

  // Group credentials by type
  const groupedCredentials = credentials.reduce(
    (acc, cred) => {
      const type = cred.credential_type;
      if (!acc[type]) {
        acc[type] = [];
      }
      acc[type].push(cred);
      return acc;
    },
    {} as Record<CredentialType, Credential[]>
  );

  return (
    <div className={className}>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
          {t.credentials.title}
        </h2>
        <Button onClick={() => setShowCreateModal(true)} variant="primary" icon={<Plus size={16} />}>
          {t.credentials.create}
        </Button>
      </div>

      {/* Filters */}
      <div className="flex gap-4 mb-6">
        <select
          value={filterType}
          onChange={(e) => setFilterType(e.target.value as CredentialType | '')}
          className="px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
        >
          <option value="">{t.credentials.allTypes}</option>
          <option value="api_key">API Key</option>
          <option value="basic_auth">Basic Auth</option>
          <option value="oauth2">OAuth2</option>
          <option value="service_account">Service Account</option>
          <option value="custom">Custom</option>
        </select>
        <input
          type="text"
          value={filterProvider}
          onChange={(e) => setFilterProvider(e.target.value)}
          placeholder={t.credentials.filterByProvider}
          className="px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
        />
      </div>

      {/* Loading state */}
      {loading && (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 mb-6">
          <p className="text-red-700 dark:text-red-300">{error}</p>
          <Button onClick={loadCredentials} variant="ghost" size="sm" className="mt-2">
            {t.common.retry}
          </Button>
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && credentials.length === 0 && (
        <div className="text-center py-12 bg-slate-50 dark:bg-slate-900 rounded-xl border-2 border-dashed border-slate-200 dark:border-slate-700">
          <div className="mx-auto w-12 h-12 bg-slate-200 dark:bg-slate-700 rounded-full flex items-center justify-center mb-3">
            <Lock size={24} className="text-slate-400 dark:text-slate-500" />
          </div>
          <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-1">
            {t.credentials.emptyTitle}
          </h3>
          <p className="text-slate-500 dark:text-slate-400 mb-4">
            {t.credentials.emptyDescription}
          </p>
          <Button onClick={() => setShowCreateModal(true)} variant="primary" icon={<Plus size={16} />}>
            {t.credentials.createFirst}
          </Button>
        </div>
      )}

      {/* Credentials list */}
      {!loading && !error && credentials.length > 0 && (
        <div className="space-y-6">
          {Object.entries(groupedCredentials).map(([type, creds]) => (
            <div key={type}>
              <h3 className="text-sm font-medium text-slate-500 dark:text-slate-400 mb-3 uppercase tracking-wide">
                {getCredentialTypeLabel(type as CredentialType)} ({creds.length})
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {creds.map((cred) => (
                  <CredentialCard
                    key={cred.id}
                    credential={cred}
                    onView={() => setViewingCredential(cred)}
                    onEdit={() => setEditingCredential(cred)}
                    onDelete={() => setDeletingCredential(cred)}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create modal */}
      <CreateCredentialModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSubmit={handleCreate}
      />

      {/* Edit modal */}
      {editingCredential && (
        <EditCredentialModal
          isOpen={true}
          credential={editingCredential}
          onClose={() => setEditingCredential(null)}
          onSubmit={handleEdit}
        />
      )}

      {/* View secrets modal */}
      {viewingCredential && (
        <ViewSecretsModal
          isOpen={true}
          credential={viewingCredential}
          onClose={() => setViewingCredential(null)}
        />
      )}

      {/* Delete confirmation modal */}
      {deletingCredential && (
        <DeleteConfirmModal
          credential={deletingCredential}
          onConfirm={() => handleDelete(deletingCredential)}
          onCancel={() => setDeletingCredential(null)}
        />
      )}
    </div>
  );
};

// Delete confirmation modal
interface DeleteConfirmModalProps {
  credential: Credential;
  onConfirm: () => void;
  onCancel: () => void;
}

const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  credential,
  onConfirm,
  onCancel,
}) => {
  const t = useTranslation();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md w-full mx-4 shadow-xl">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          {t.credentials.deleteTitle}
        </h3>
        <p className="text-slate-600 dark:text-slate-400 mb-4">
          {t.credentials.deleteConfirmation.replace('{name}', credential.name)}
        </p>
        <div className="flex justify-end gap-3">
          <Button onClick={onCancel} variant="secondary">
            {t.common.cancel}
          </Button>
          <Button onClick={onConfirm} variant="primary" className="bg-red-500 hover:bg-red-600">
            {t.common.delete}
          </Button>
        </div>
      </div>
    </div>
  );
};
