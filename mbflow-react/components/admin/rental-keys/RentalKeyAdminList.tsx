/**
 * RentalKeyAdminList component
 * Admin panel for managing all rental keys across users.
 */

import React, { useState, useEffect, useRef, useMemo } from 'react';
import {
  Key,
  AlertCircle,
  RefreshCw,
  Plus,
  Edit2,
  Trash2,
  RotateCw,
  ChevronLeft,
  ChevronRight,
  User,
  Search,
  ChevronDown,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui';
import { ConfirmModal } from '@/components/ui/ConfirmModal';
import { CreateRentalKeyModal } from './CreateRentalKeyModal';
import { EditRentalKeyModal } from './EditRentalKeyModal';
import { RotateKeyModal } from './RotateKeyModal';
import {
  AdminRentalKey,
  LLMProviderType,
  ResourceStatus,
  RentalKeyFilter,
  rentalKeyAdminApi,
  getProviderLabel,
  getStatusColor,
  formatTokenCount,
  LLM_PROVIDERS,
} from '@/services/rentalKeyService';
import { formatShortDate } from '@/utils/formatters';
import { useTranslation } from '@/store/translations';
import { authService } from '@/services/authService';
import type { User as UserType } from '@/types/auth';

interface RentalKeyAdminListProps {
  className?: string;
}

export const RentalKeyAdminList: React.FC<RentalKeyAdminListProps> = ({ className }) => {
  const t = useTranslation();
  const [rentalKeys, setRentalKeys] = useState<AdminRentalKey[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Users for filter and display
  const [users, setUsers] = useState<UserType[]>([]);
  const [usersMap, setUsersMap] = useState<Map<string, UserType>>(new Map());
  const [usersLoading, setUsersLoading] = useState(false);
  const [userSearch, setUserSearch] = useState('');
  const [isUserDropdownOpen, setIsUserDropdownOpen] = useState(false);
  const [selectedFilterUser, setSelectedFilterUser] = useState<UserType | null>(null);
  const userDropdownRef = useRef<HTMLDivElement>(null);

  // Filters
  const [filter, setFilter] = useState<RentalKeyFilter>({
    limit: 20,
    offset: 0,
  });

  // Modals
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [rotateModalOpen, setRotateModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [selectedKey, setSelectedKey] = useState<AdminRentalKey | null>(null);

  // Load users on mount
  useEffect(() => {
    loadUsers();
  }, []);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (userDropdownRef.current && !userDropdownRef.current.contains(e.target as Node)) {
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
      const map = new Map<string, UserType>();
      response.data.forEach((user) => map.set(user.id, user));
      setUsersMap(map);
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setUsersLoading(false);
    }
  };

  // Filter users for dropdown
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

  const handleSelectFilterUser = (user: UserType | null) => {
    setSelectedFilterUser(user);
    setFilter((prev) => ({
      ...prev,
      owner_id: user?.id || undefined,
      offset: 0,
    }));
    setIsUserDropdownOpen(false);
    setUserSearch('');
  };

  const fetchRentalKeys = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await rentalKeyAdminApi.listAllRentalKeys(filter);
      setRentalKeys(response.data.rental_keys || []);
      setTotal(response.data.total || 0);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load rental keys');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRentalKeys();
  }, [filter]);

  const handleFilterChange = (key: keyof RentalKeyFilter, value: any) => {
    setFilter((prev) => ({
      ...prev,
      [key]: value || undefined,
      offset: 0, // Reset pagination on filter change
    }));
  };

  const handlePrevPage = () => {
    setFilter((prev) => ({
      ...prev,
      offset: Math.max(0, (prev.offset || 0) - (prev.limit || 20)),
    }));
  };

  const handleNextPage = () => {
    const currentOffset = filter.offset || 0;
    const limit = filter.limit || 20;
    if (currentOffset + limit < total) {
      setFilter((prev) => ({
        ...prev,
        offset: currentOffset + limit,
      }));
    }
  };

  const handleEdit = (key: AdminRentalKey) => {
    setSelectedKey(key);
    setEditModalOpen(true);
  };

  const handleRotate = (key: AdminRentalKey) => {
    setSelectedKey(key);
    setRotateModalOpen(true);
  };

  const handleDelete = (key: AdminRentalKey) => {
    setSelectedKey(key);
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    if (!selectedKey) return;
    try {
      await rentalKeyAdminApi.deleteRentalKey(selectedKey.id);
      fetchRentalKeys();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete rental key');
    }
    setDeleteModalOpen(false);
    setSelectedKey(null);
  };

  return (
    <div className={className}>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-bold text-slate-900 dark:text-white flex items-center">
          <Key size={24} className="mr-2" />
          {t.admin?.rentalKeysManagement || 'Rental Keys Management'}
        </h2>
        <div className="flex items-center gap-2">
          <Button
            onClick={fetchRentalKeys}
            variant="ghost"
            size="sm"
            icon={<RefreshCw size={14} />}
            disabled={loading}
          >
            {t.common?.refresh || 'Refresh'}
          </Button>
          <Button
            onClick={() => setCreateModalOpen(true)}
            size="sm"
            icon={<Plus size={14} />}
          >
            {t.admin?.createKey || 'Create Key'}
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 mb-4">
        <select
          value={filter.provider || ''}
          onChange={(e) => handleFilterChange('provider', e.target.value)}
          className="text-sm border border-slate-200 dark:border-slate-700 rounded-lg px-3 py-1.5 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
        >
          <option value="">{t.admin?.allProviders || 'All Providers'}</option>
          {LLM_PROVIDERS.map((provider) => (
            <option key={provider.value} value={provider.value}>
              {provider.label}
            </option>
          ))}
        </select>
        <select
          value={filter.status || ''}
          onChange={(e) => handleFilterChange('status', e.target.value)}
          className="text-sm border border-slate-200 dark:border-slate-700 rounded-lg px-3 py-1.5 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
        >
          <option value="">{t.admin?.allStatuses || 'All Statuses'}</option>
          <option value="active">{t.status?.active || 'Active'}</option>
          <option value="suspended">{t.status?.suspended || 'Suspended'}</option>
          <option value="deleted">{t.status?.deleted || 'Deleted'}</option>
        </select>

        {/* User filter dropdown */}
        <div ref={userDropdownRef} className="relative">
          <button
            type="button"
            onClick={() => setIsUserDropdownOpen(!isUserDropdownOpen)}
            className={`flex items-center gap-2 text-sm border rounded-lg px-3 py-1.5 bg-white dark:bg-slate-800 min-w-[200px] transition-colors ${
              isUserDropdownOpen
                ? 'border-blue-500 ring-2 ring-blue-500/20'
                : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
            }`}
          >
            {selectedFilterUser ? (
              <>
                <div className="w-5 h-5 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                  {selectedFilterUser.username.slice(0, 1).toUpperCase()}
                </div>
                <span className="text-slate-900 dark:text-white truncate flex-1 text-left">
                  {selectedFilterUser.full_name || selectedFilterUser.username}
                </span>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSelectFilterUser(null);
                  }}
                  className="p-0.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                >
                  <X size={14} className="text-slate-400" />
                </button>
              </>
            ) : (
              <>
                <User size={14} className="text-slate-400" />
                <span className="text-slate-400 dark:text-slate-500 flex-1 text-left">
                  {t.admin?.allOwners || 'All Owners'}
                </span>
                <ChevronDown size={14} className="text-slate-400" />
              </>
            )}
          </button>

          {isUserDropdownOpen && (
            <div className="absolute z-50 w-72 mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg max-h-72 overflow-hidden">
              <div className="p-2 border-b border-slate-200 dark:border-slate-700">
                <div className="relative">
                  <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
                  <input
                    type="text"
                    value={userSearch}
                    onChange={(e) => setUserSearch(e.target.value)}
                    placeholder={t.admin?.searchUsers || 'Search users...'}
                    className="w-full pl-8 pr-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    autoFocus
                  />
                </div>
              </div>
              <div className="max-h-52 overflow-y-auto">
                {/* All owners option */}
                <button
                  type="button"
                  onClick={() => handleSelectFilterUser(null)}
                  className={`w-full px-3 py-2 flex items-center gap-2 hover:bg-slate-50 dark:hover:bg-slate-700/50 text-sm ${
                    !selectedFilterUser ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                  }`}
                >
                  <User size={14} className="text-slate-400" />
                  <span className="text-slate-600 dark:text-slate-300">
                    {t.admin?.allOwners || 'All Owners'}
                  </span>
                </button>
                {usersLoading ? (
                  <div className="p-3 text-center text-slate-500 dark:text-slate-400 text-sm">
                    <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-1" />
                    {t.common?.loading || 'Loading...'}
                  </div>
                ) : filteredUsers.length === 0 ? (
                  <div className="p-3 text-center text-slate-500 dark:text-slate-400 text-sm">
                    {t.admin?.noUsersFound || 'No users found'}
                  </div>
                ) : (
                  filteredUsers.map((user) => (
                    <button
                      key={user.id}
                      type="button"
                      onClick={() => handleSelectFilterUser(user)}
                      className={`w-full px-3 py-2 flex items-center gap-2 hover:bg-slate-50 dark:hover:bg-slate-700/50 ${
                        selectedFilterUser?.id === user.id ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                      }`}
                    >
                      <div className="w-6 h-6 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                        {user.username.slice(0, 2).toUpperCase()}
                      </div>
                      <div className="flex-1 min-w-0 text-left">
                        <div className="text-sm font-medium text-slate-900 dark:text-white truncate">
                          {user.full_name || user.username}
                        </div>
                        <div className="text-xs text-slate-500 dark:text-slate-400 truncate">
                          @{user.username} • {user.email}
                        </div>
                      </div>
                    </button>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-center text-red-700 dark:text-red-400">
          <AlertCircle size={16} className="mr-2" />
          {error}
        </div>
      )}

      {/* Table */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-slate-50 dark:bg-slate-800">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.common?.name || 'Name'}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.admin?.owner || 'Owner'}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.admin?.provider || 'Provider'}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.common?.status || 'Status'}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.admin?.usage || 'Usage'}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.common?.created || 'Created'}
                </th>
                <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  {t.common?.actions || 'Actions'}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
              {loading && rentalKeys.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                  </td>
                </tr>
              ) : rentalKeys.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-slate-500 dark:text-slate-400">
                    {t.admin?.noRentalKeys || 'No rental keys found'}
                  </td>
                </tr>
              ) : (
                rentalKeys.map((key) => (
                  <RentalKeyRow
                    key={key.id}
                    rentalKey={key}
                    owner={usersMap.get(key.owner_id)}
                    onEdit={() => handleEdit(key)}
                    onRotate={() => handleRotate(key)}
                    onDelete={() => handleDelete(key)}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {total > 0 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-slate-200 dark:border-slate-700">
            <span className="text-sm text-slate-500 dark:text-slate-400">
              {t.admin?.showingOf || 'Showing'} {(filter.offset || 0) + 1} - {Math.min((filter.offset || 0) + (filter.limit || 20), total)} {t.admin?.of || 'of'} {total}
            </span>
            <div className="flex items-center gap-2">
              <Button
                onClick={handlePrevPage}
                disabled={(filter.offset || 0) === 0}
                variant="outline"
                size="sm"
                icon={<ChevronLeft size={14} />}
              />
              <Button
                onClick={handleNextPage}
                disabled={(filter.offset || 0) + (filter.limit || 20) >= total}
                variant="outline"
                size="sm"
                icon={<ChevronRight size={14} />}
              />
            </div>
          </div>
        )}
      </div>

      {/* Modals */}
      <CreateRentalKeyModal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onCreated={fetchRentalKeys}
      />

      {selectedKey && (
        <>
          <EditRentalKeyModal
            isOpen={editModalOpen}
            onClose={() => {
              setEditModalOpen(false);
              setSelectedKey(null);
            }}
            onUpdated={fetchRentalKeys}
            rentalKey={selectedKey}
          />
          <RotateKeyModal
            isOpen={rotateModalOpen}
            onClose={() => {
              setRotateModalOpen(false);
              setSelectedKey(null);
            }}
            onRotated={fetchRentalKeys}
            rentalKey={selectedKey}
          />
        </>
      )}

      <ConfirmModal
        isOpen={deleteModalOpen}
        onClose={() => {
          setDeleteModalOpen(false);
          setSelectedKey(null);
        }}
        onConfirm={confirmDelete}
        title={t.admin?.deleteRentalKey || 'Delete Rental Key'}
        message={`${t.admin?.deleteRentalKeyConfirm || 'Are you sure you want to delete'} "${selectedKey?.name}"?`}
        confirmText={t.common?.delete || 'Delete'}
        variant="danger"
      />
    </div>
  );
};

interface RentalKeyRowProps {
  rentalKey: AdminRentalKey;
  owner?: UserType;
  onEdit: () => void;
  onRotate: () => void;
  onDelete: () => void;
}

const RentalKeyRow: React.FC<RentalKeyRowProps> = ({
  rentalKey,
  owner,
  onEdit,
  onRotate,
  onDelete,
}) => {
  const statusColors: Record<string, string> = {
    active: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
    suspended: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400',
    deleted: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400',
  };

  return (
    <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
      <td className="px-4 py-3">
        <div className="font-medium text-slate-900 dark:text-white">
          {rentalKey.name}
        </div>
        {rentalKey.description && (
          <div className="text-xs text-slate-500 dark:text-slate-400 truncate max-w-xs">
            {rentalKey.description}
          </div>
        )}
      </td>
      <td className="px-4 py-3">
        {owner ? (
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
              {owner.username.slice(0, 2).toUpperCase()}
            </div>
            <div className="min-w-0">
              <div className="text-sm font-medium text-slate-900 dark:text-white truncate max-w-[120px]">
                {owner.full_name || owner.username}
              </div>
              <div className="text-xs text-slate-500 dark:text-slate-400 truncate max-w-[120px]">
                @{owner.username}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex items-center text-sm text-slate-600 dark:text-slate-400">
            <User size={14} className="mr-1" />
            <span className="font-mono text-xs truncate max-w-[120px]" title={rentalKey.owner_id}>
              {rentalKey.owner_id.slice(0, 8)}...
            </span>
          </div>
        )}
      </td>
      <td className="px-4 py-3">
        <span className="text-sm text-slate-900 dark:text-white">
          {getProviderLabel(rentalKey.provider)}
        </span>
      </td>
      <td className="px-4 py-3">
        <span className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${statusColors[rentalKey.status] || 'bg-gray-100 text-gray-800'}`}>
          {rentalKey.status}
        </span>
      </td>
      <td className="px-4 py-3">
        <div className="text-sm">
          <span className="text-slate-900 dark:text-white">
            {formatTokenCount(rentalKey.total_requests)}
          </span>
          <span className="text-slate-500 dark:text-slate-400 ml-1">req</span>
        </div>
        <div className="text-xs text-slate-500 dark:text-slate-400">
          ${rentalKey.total_cost.toFixed(2)}
        </div>
      </td>
      <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
        {formatShortDate(rentalKey.created_at)}
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center justify-end gap-1">
          <button
            onClick={onEdit}
            className="p-1.5 text-slate-500 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
            title="Edit"
          >
            <Edit2 size={14} />
          </button>
          <button
            onClick={onRotate}
            className="p-1.5 text-slate-500 hover:text-orange-600 hover:bg-orange-50 dark:hover:bg-orange-900/20 rounded transition-colors"
            title="Rotate API Key"
          >
            <RotateCw size={14} />
          </button>
          <button
            onClick={onDelete}
            className="p-1.5 text-slate-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
            title="Delete"
          >
            <Trash2 size={14} />
          </button>
        </div>
      </td>
    </tr>
  );
};

export default RentalKeyAdminList;
