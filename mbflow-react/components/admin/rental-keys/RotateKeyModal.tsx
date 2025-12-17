/**
 * RotateKeyModal component
 * Admin modal for rotating a rental key's API key.
 */

import React, { useState } from 'react';
import { X, RotateCw, AlertCircle, AlertTriangle } from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import {
  AdminRentalKey,
  rentalKeyAdminApi,
  getProviderLabel,
} from '@/services/rentalKeyService';
import { useTranslation } from '@/store/translations';

interface RotateKeyModalProps {
  isOpen: boolean;
  onClose: () => void;
  onRotated: () => void;
  rentalKey: AdminRentalKey;
}

export const RotateKeyModal: React.FC<RotateKeyModalProps> = ({
  isOpen,
  onClose,
  onRotated,
  rentalKey,
}) => {
  const t = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newAPIKey, setNewAPIKey] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newAPIKey.trim()) {
      setError('API key is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await rentalKeyAdminApi.rotateAPIKey(rentalKey.id, { new_api_key: newAPIKey });
      onRotated();
      onClose();
      setNewAPIKey('');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to rotate API key');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setNewAPIKey('');
    setError(null);
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="" size="md">
      <form onSubmit={handleSubmit} className="p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center">
            <div className="p-2 bg-orange-50 dark:bg-orange-900/20 rounded-lg mr-3">
              <RotateCw size={20} className="text-orange-600 dark:text-orange-400" />
            </div>
            <h2 className="text-xl font-bold text-slate-900 dark:text-white">
              {t.admin?.rotateAPIKey || 'Rotate API Key'}
            </h2>
          </div>
          <button
            type="button"
            onClick={handleClose}
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
          >
            <X size={20} className="text-slate-500" />
          </button>
        </div>

        {/* Warning */}
        <div className="mb-4 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg flex items-start text-yellow-700 dark:text-yellow-400">
          <AlertTriangle size={16} className="mr-2 flex-shrink-0 mt-0.5" />
          <div className="text-sm">
            <p className="font-medium mb-1">
              {t.admin?.rotateKeyWarningTitle || 'Warning'}
            </p>
            <p>
              {t.admin?.rotateKeyWarning || 'Rotating the API key will immediately invalidate the old key. Any workflows using this rental key will start using the new key on their next execution.'}
            </p>
          </div>
        </div>

        {/* Error message */}
        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-center text-red-700 dark:text-red-400">
            <AlertCircle size={16} className="mr-2 flex-shrink-0" />
            {error}
          </div>
        )}

        {/* Key info */}
        <div className="mb-4 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg">
          <div className="text-sm text-slate-500 dark:text-slate-400 mb-1">
            {t.admin?.rotatingKeyFor || 'Rotating key for'}:
          </div>
          <div className="font-medium text-slate-900 dark:text-white">
            {rentalKey.name}
          </div>
          <div className="text-sm text-slate-500 dark:text-slate-400">
            {getProviderLabel(rentalKey.provider)}
          </div>
        </div>

        {/* New API Key */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            {t.admin?.newAPIKey || 'New API Key'} *
          </label>
          <input
            type="password"
            value={newAPIKey}
            onChange={(e) => setNewAPIKey(e.target.value)}
            required
            placeholder="Enter the new API key"
            className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono"
          />
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            {t.admin?.newAPIKeyHint || 'This key will be encrypted and never shown again.'}
          </p>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-slate-200 dark:border-slate-700">
          <Button type="button" variant="outline" onClick={handleClose}>
            {t.common?.cancel || 'Cancel'}
          </Button>
          <Button type="submit" loading={loading} variant="danger">
            {t.admin?.rotateKey || 'Rotate Key'}
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default RotateKeyModal;
