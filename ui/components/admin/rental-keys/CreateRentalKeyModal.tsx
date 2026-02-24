/**
 * CreateRentalKeyModal component
 * Admin modal for creating new rental keys.
 */

import React, { useState, useEffect, useRef, useMemo } from 'react';
import { X, Key, AlertCircle, Search, ChevronDown, Check } from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import {
  CreateRentalKeyRequest,
  LLMProviderType,
  ProvisionerType,
  rentalKeyAdminApi,
  LLM_PROVIDERS,
  PROVISIONER_TYPES,
} from '@/services/rentalKeyService';
import { useTranslation } from '@/store/translations';
import { authService } from '@/services/authService';
import type { User } from '@/types/auth';
import { getErrorMessage } from '@/lib/api';

interface CreateRentalKeyModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated: () => void;
}

export const CreateRentalKeyModal: React.FC<CreateRentalKeyModalProps> = ({
  isOpen,
  onClose,
  onCreated,
}) => {
  const t = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Users state
  const [users, setUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [userSearch, setUserSearch] = useState('');
  const [isUserDropdownOpen, setIsUserDropdownOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const [formData, setFormData] = useState<CreateRentalKeyRequest>({
    owner_id: '',
    name: '',
    description: '',
    provider: 'openai',
    api_key: '',
    daily_request_limit: undefined,
    monthly_token_limit: undefined,
    provisioner_type: 'manual',
  });

  // Load users when modal opens
  useEffect(() => {
    if (isOpen) {
      loadUsers();
    }
  }, [isOpen]);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsUserDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const loadUsers = async () => {
    setUsersLoading(true);
    try {
      const response = await authService.listUsers(500, 0);
      setUsers(response.data);
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setUsersLoading(false);
    }
  };

  // Filter users based on search
  const filteredUsers = useMemo(() => {
    if (!userSearch.trim()) return users;
    const search = userSearch.toLowerCase();
    return users.filter(
      (user) =>
        user.username.toLowerCase().includes(search) ||
        user.email.toLowerCase().includes(search) ||
        (user.full_name && user.full_name.toLowerCase().includes(search))
    );
  }, [users, userSearch]);

  const handleSelectUser = (user: User) => {
    setSelectedUser(user);
    setFormData((prev) => ({ ...prev, owner_id: user.id }));
    setIsUserDropdownOpen(false);
    setUserSearch('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      await rentalKeyAdminApi.createRentalKey(formData);
      onCreated();
      onClose();
      // Reset form
      setFormData({
        owner_id: '',
        name: '',
        description: '',
        provider: 'openai',
        api_key: '',
        daily_request_limit: undefined,
        monthly_token_limit: undefined,
        provisioner_type: 'manual',
      });
      setSelectedUser(null);
      setUserSearch('');
    } catch (error: unknown) {
      setError(getErrorMessage(error));
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
              {t.admin?.createRentalKey || 'Create Rental Key'}
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
          {/* Owner User Select */}
          <div ref={dropdownRef} className="relative">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.admin?.ownerUser || 'Owner'} *
            </label>
            <button
              type="button"
              onClick={() => setIsUserDropdownOpen(!isUserDropdownOpen)}
              className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-slate-800 text-left flex items-center justify-between transition-colors ${
                isUserDropdownOpen
                  ? 'border-blue-500 ring-2 ring-blue-500'
                  : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
              }`}
            >
              {selectedUser ? (
                <div className="flex items-center gap-2 min-w-0">
                  <div className="w-6 h-6 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                    {selectedUser.username.slice(0, 2).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <span className="text-slate-900 dark:text-white truncate block">
                      {selectedUser.full_name || selectedUser.username}
                    </span>
                    <span className="text-xs text-slate-500 dark:text-slate-400 truncate block">
                      {selectedUser.email}
                    </span>
                  </div>
                </div>
              ) : (
                <span className="text-slate-400 dark:text-slate-500">
                  {t.admin?.selectUser || 'Select user...'}
                </span>
              )}
              <ChevronDown
                size={16}
                className={`text-slate-400 flex-shrink-0 transition-transform ${
                  isUserDropdownOpen ? 'rotate-180' : ''
                }`}
              />
            </button>

            {/* Dropdown */}
            {isUserDropdownOpen && (
              <div className="absolute z-50 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg max-h-72 overflow-hidden">
                {/* Search input */}
                <div className="p-2 border-b border-slate-200 dark:border-slate-700">
                  <div className="relative">
                    <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                    <input
                      type="text"
                      value={userSearch}
                      onChange={(e) => setUserSearch(e.target.value)}
                      placeholder={t.admin?.searchUsers || 'Search users...'}
                      className="w-full pl-9 pr-3 py-2 text-sm border border-slate-200 dark:border-slate-700 rounded-md bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      autoFocus
                    />
                  </div>
                </div>

                {/* User list */}
                <div className="max-h-52 overflow-y-auto">
                  {usersLoading ? (
                    <div className="p-4 text-center text-slate-500 dark:text-slate-400">
                      <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                      {t.common?.loading || 'Loading...'}
                    </div>
                  ) : filteredUsers.length === 0 ? (
                    <div className="p-4 text-center text-slate-500 dark:text-slate-400">
                      {t.admin?.noUsersFound || 'No users found'}
                    </div>
                  ) : (
                    filteredUsers.map((user) => (
                      <button
                        key={user.id}
                        type="button"
                        onClick={() => handleSelectUser(user)}
                        className={`w-full px-3 py-2 flex items-center gap-3 hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors ${
                          selectedUser?.id === user.id ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                        }`}
                      >
                        <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center text-white text-sm font-medium flex-shrink-0">
                          {user.username.slice(0, 2).toUpperCase()}
                        </div>
                        <div className="flex-1 min-w-0 text-left">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-slate-900 dark:text-white truncate">
                              {user.full_name || user.username}
                            </span>
                            {user.is_admin && (
                              <span className="px-1.5 py-0.5 text-xs font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300 rounded">
                                Admin
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                            <span className="truncate">@{user.username}</span>
                            <span className="text-slate-300 dark:text-slate-600">•</span>
                            <span className="truncate">{user.email}</span>
                          </div>
                        </div>
                        {selectedUser?.id === user.id && (
                          <Check size={16} className="text-blue-500 flex-shrink-0" />
                        )}
                      </button>
                    ))
                  )}
                </div>
              </div>
            )}

            {/* Hidden input for form validation */}
            <input
              type="hidden"
              name="owner_id"
              value={formData.owner_id}
              required
            />
          </div>

          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.common?.name || 'Name'} *
            </label>
            <input
              type="text"
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
              maxLength={255}
              placeholder="Rental key name"
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
              placeholder="Optional description"
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
            />
          </div>

          {/* Provider */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.admin?.provider || 'Provider'} *
            </label>
            <select
              name="provider"
              value={formData.provider}
              onChange={handleChange}
              required
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {LLM_PROVIDERS.map((provider) => (
                <option key={provider.value} value={provider.value}>
                  {provider.label}
                </option>
              ))}
            </select>
          </div>

          {/* API Key */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.admin?.apiKey || 'API Key'} *
            </label>
            <input
              type="password"
              name="api_key"
              value={formData.api_key}
              onChange={handleChange}
              required
              placeholder="The actual API key (will be encrypted)"
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono"
            />
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t.admin?.apiKeyHint || 'This key will be encrypted and never shown again.'}
            </p>
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

          {/* Provisioner Type */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t.admin?.provisionerType || 'Provisioner Type'}
            </label>
            <select
              name="provisioner_type"
              value={formData.provisioner_type}
              onChange={handleChange}
              className="w-full px-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {PROVISIONER_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-slate-200 dark:border-slate-700">
          <Button type="button" variant="outline" onClick={onClose}>
            {t.common?.cancel || 'Cancel'}
          </Button>
          <Button type="submit" loading={loading}>
            {t.common?.create || 'Create'}
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateRentalKeyModal;
