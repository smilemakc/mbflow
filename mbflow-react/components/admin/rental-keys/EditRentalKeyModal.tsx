/**
 * EditRentalKeyModal component
 * Admin modal for editing rental key properties.
 */

import React, { useState } from 'react';
import { X, Key, AlertCircle } from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import {
  AdminRentalKey,
  UpdateRentalKeyRequest,
  ResourceStatus,
  rentalKeyAdminApi,
} from '@/services/rentalKeyService';
import { useTranslation } from '@/store/translations';

interface EditRentalKeyModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdated: () => void;
  rentalKey: AdminRentalKey;
}

export const EditRentalKeyModal: React.FC<EditRentalKeyModalProps> = ({
  isOpen,
  onClose,
  onUpdated,
  rentalKey,
}) => {
  const t = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState<{
    name: string;
    description: string;
    status: ResourceStatus;
    daily_request_limit: number | undefined;
    monthly_token_limit: number | undefined;
  }>({
    name: rentalKey.name,
    description: rentalKey.description || '',
    status: rentalKey.status,
    daily_request_limit: rentalKey.daily_request_limit,
    monthly_token_limit: rentalKey.monthly_token_limit,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const updateReq: UpdateRentalKeyRequest = {};
    if (formData.name !== rentalKey.name) {
      updateReq.name = formData.name;
    }
    if (formData.description !== (rentalKey.description || '')) {
      updateReq.description = formData.description;
    }
    if (formData.status !== rentalKey.status) {
      updateReq.status = formData.status;
    }
    if (formData.daily_request_limit !== rentalKey.daily_request_limit) {
      updateReq.daily_request_limit = formData.daily_request_limit;
    }
    if (formData.monthly_token_limit !== rentalKey.monthly_token_limit) {
      updateReq.monthly_token_limit = formData.monthly_token_limit;
    }

    try {
      await rentalKeyAdminApi.updateRentalKey(rentalKey.id, updateReq);
      onUpdated();
      onClose();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to update rental key');
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value, type } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === 'number' ? (value ? Number(value) : undefined) : value,
    }));
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="" size="lg">
      <form onSubmit={handleSubmit} className="p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center">
            <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg mr-3">
              <Key size={20} className="text-blue-600 dark:text-blue-400" />
            </div>
            <h2 className="text-xl font-bold text-slate-900 dark:text-white">
              {t.admin?.editRentalKey || 'Edit Rental Key'}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
          >
            <X size={20} className="text-slate-500" />
          </button>
        </div>

        {/* Error message */}
        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-center text-red-700 dark:text-red-400">
            <AlertCircle size={16} className="mr-2 flex-shrink-0" />
            {error}
          </div>
        )}

        <div className="space-y-4">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.common?.name || 'Name'}
            </label>
            <input
              type="text"
              name="name"
              value={formData.name}
              onChange={handleChange}
              maxLength={255}
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.common?.description || 'Description'}
            </label>
            <textarea
              name="description"
              value={formData.description}
              onChange={handleChange}
              maxLength={1000}
              rows={2}
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
            />
          </div>

          {/* Status */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.common?.status || 'Status'}
            </label>
            <select
              name="status"
              value={formData.status}
              onChange={handleChange}
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="active">{t.status?.active || 'Active'}</option>
              <option value="suspended">{t.status?.suspended || 'Suspended'}</option>
            </select>
          </div>

          {/* Limits */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t.admin?.dailyRequestLimit || 'Daily Request Limit'}
              </label>
              <input
                type="number"
                name="daily_request_limit"
                value={formData.daily_request_limit || ''}
                onChange={handleChange}
                min={1}
                placeholder="Unlimited"
                className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t.admin?.monthlyTokenLimit || 'Monthly Token Limit'}
              </label>
              <input
                type="number"
                name="monthly_token_limit"
                value={formData.monthly_token_limit || ''}
                onChange={handleChange}
                min={1}
                placeholder="Unlimited"
                className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-slate-200 dark:border-slate-700">
          <Button type="button" variant="outline" onClick={onClose}>
            {t.common?.cancel || 'Cancel'}
          </Button>
          <Button type="submit" loading={loading}>
            {t.common?.save || 'Save'}
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditRentalKeyModal;
