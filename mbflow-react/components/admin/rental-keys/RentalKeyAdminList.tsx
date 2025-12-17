/**
 * RentalKeyAdminList component
 * Admin panel for managing all rental keys across users.
 */

import React, { useState, useEffect, useMemo } from 'react';
import { Key, RefreshCw, Plus, Edit2, Trash2, RotateCw, User } from 'lucide-react';
import { Button } from '@/components/ui';
import { DataTable, Column } from '@/components/ui/DataTable';
import { FilterBar, FilterSelect } from '@/components/ui/FilterBar';
import { UserSelector } from '@/components/admin/UserSelector';
import { ConfirmModal } from '@/components/ui/ConfirmModal';
import { useTableData } from '@/hooks/useTableData';
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
  formatTokenCount,
  LLM_PROVIDERS,
} from '@/services/rentalKeyService';
import { formatShortDate } from '@/utils/formatters';
import { useTranslation } from '@/store/translations';
import { authService } from '@/services/authService';
import { getErrorMessage } from '@/lib/api';
import { toast } from '@/lib/toast';
import type { User as UserType } from '@/types/auth';

interface RentalKeyAdminListProps {
  className?: string;
}

export const RentalKeyAdminList: React.FC<RentalKeyAdminListProps> = ({ className }) => {
  const t = useTranslation();

  const [users, setUsers] = useState<UserType[]>([]);
  const [usersMap, setUsersMap] = useState<Map<string, UserType>>(new Map());
  const [usersLoading, setUsersLoading] = useState(false);
  const [selectedFilterUser, setSelectedFilterUser] = useState<UserType | null>(null);

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [rotateModalOpen, setRotateModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [selectedKey, setSelectedKey] = useState<AdminRentalKey | null>(null);

  const table = useTableData<AdminRentalKey, RentalKeyFilter>({
    fetchFn: async ({ limit, offset, filters }) => {
      const response = await rentalKeyAdminApi.listAllRentalKeys({ limit, offset, ...filters });
      return {
        items: response.data.rental_keys || [],
        total: response.data.total || 0,
      };
    },
    initialLimit: 20,
    initialFilters: {},
  });

  useEffect(() => {
    loadUsers();
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

  const handleUserFilterChange = (user: UserType | null) => {
    setSelectedFilterUser(user);
    table.setFilters({
      ...table.filters,
      owner_id: user?.id,
    });
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
      table.refresh();
    } catch (error: unknown) {
      console.error('Failed to delete rental key:', error);
      toast.error('Delete Failed', getErrorMessage(error));
    }
    setDeleteModalOpen(false);
    setSelectedKey(null);
  };

  const statusColors: Record<string, string> = {
    active: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
    suspended: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400',
    deleted: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400',
  };

  const columns: Column<AdminRentalKey>[] = [
    {
      key: 'name',
      header: t.common?.name || 'Name',
      render: (key) => (
        <div>
          <div className="font-medium text-slate-900 dark:text-white">{key.name}</div>
          {key.description && (
            <div className="text-xs text-slate-500 dark:text-slate-400 truncate max-w-xs">
              {key.description}
            </div>
          )}
        </div>
      ),
    },
    {
      key: 'owner_id',
      header: t.admin?.owner || 'Owner',
      render: (key) => {
        const owner = usersMap.get(key.owner_id);
        if (owner) {
          return (
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
          );
        }
        return (
          <div className="flex items-center text-sm text-slate-600 dark:text-slate-400">
            <User size={14} className="mr-1" />
            <span className="font-mono text-xs truncate max-w-[120px]" title={key.owner_id}>
              {key.owner_id.slice(0, 8)}...
            </span>
          </div>
        );
      },
    },
    {
      key: 'provider',
      header: t.admin?.provider || 'Provider',
      render: (key) => (
        <span className="text-sm text-slate-900 dark:text-white">
          {getProviderLabel(key.provider)}
        </span>
      ),
    },
    {
      key: 'status',
      header: t.common?.status || 'Status',
      render: (key) => (
        <span
          className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${
            statusColors[key.status] || 'bg-gray-100 text-gray-800'
          }`}
        >
          {key.status}
        </span>
      ),
    },
    {
      key: 'usage',
      header: t.admin?.usage || 'Usage',
      render: (key) => (
        <div>
          <div className="text-sm">
            <span className="text-slate-900 dark:text-white">
              {formatTokenCount(key.total_requests)}
            </span>
            <span className="text-slate-500 dark:text-slate-400 ml-1">req</span>
          </div>
          <div className="text-xs text-slate-500 dark:text-slate-400">
            ${key.total_cost.toFixed(2)}
          </div>
        </div>
      ),
    },
    {
      key: 'created_at',
      header: t.common?.created || 'Created',
      render: (key) => (
        <span className="text-sm text-slate-500 dark:text-slate-400">
          {formatShortDate(key.created_at)}
        </span>
      ),
    },
  ];

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-bold text-slate-900 dark:text-white flex items-center">
          <Key size={24} className="mr-2" />
          {t.admin?.rentalKeysManagement || 'Rental Keys Management'}
        </h2>
        <div className="flex items-center gap-2">
          <Button
            onClick={table.refresh}
            variant="ghost"
            size="sm"
            icon={<RefreshCw size={14} />}
            disabled={table.loading}
          >
            {t.common?.refresh || 'Refresh'}
          </Button>
          <Button onClick={() => setCreateModalOpen(true)} size="sm" icon={<Plus size={14} />}>
            {t.admin?.createKey || 'Create Key'}
          </Button>
        </div>
      </div>

      <FilterBar className="mb-4">
        <FilterSelect
          value={(table.filters.provider as string) || ''}
          onChange={(value) =>
            table.setFilters({
              ...table.filters,
              provider: (value as LLMProviderType) || undefined,
            })
          }
          options={[
            { label: t.admin?.allProviders || 'All Providers', value: '' },
            ...LLM_PROVIDERS.map((p) => ({ label: p.label, value: p.value })),
          ]}
        />
        <FilterSelect
          value={(table.filters.status as string) || ''}
          onChange={(value) =>
            table.setFilters({
              ...table.filters,
              status: value as ResourceStatus | undefined,
            })
          }
          options={[
            { label: t.admin?.allStatuses || 'All Statuses', value: '' },
            { label: t.status?.active || 'Active', value: 'active' },
            { label: t.status?.suspended || 'Suspended', value: 'suspended' },
            { label: t.status?.deleted || 'Deleted', value: 'deleted' },
          ]}
        />
        <UserSelector
          value={selectedFilterUser}
          onChange={handleUserFilterChange}
          users={users}
          loading={usersLoading}
          placeholder={t.admin?.allOwners || 'All Owners'}
          className="w-64"
          showClearButton={true}
        />
      </FilterBar>

      <DataTable
        data={table.items}
        columns={columns}
        keyExtractor={(key) => key.id}
        loading={table.loading}
        error={table.error}
        emptyIcon={Key}
        emptyTitle={t.admin?.noRentalKeys || 'No rental keys found'}
        emptyDescription={t.admin?.createFirstKey || 'Create your first rental key to get started'}
        actions={(key) => (
          <>
            <button
              onClick={() => handleEdit(key)}
              className="p-1.5 text-slate-500 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
              title="Edit"
            >
              <Edit2 size={14} />
            </button>
            <button
              onClick={() => handleRotate(key)}
              className="p-1.5 text-slate-500 hover:text-orange-600 hover:bg-orange-50 dark:hover:bg-orange-900/20 rounded transition-colors"
              title="Rotate API Key"
            >
              <RotateCw size={14} />
            </button>
            <button
              onClick={() => handleDelete(key)}
              className="p-1.5 text-slate-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
              title="Delete"
            >
              <Trash2 size={14} />
            </button>
          </>
        )}
        pagination={{
          offset: table.offset,
          limit: table.limit,
          total: table.total,
          onOffsetChange: table.setOffset,
        }}
      />

      <CreateRentalKeyModal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onCreated={table.refresh}
      />

      {selectedKey && (
        <>
          <EditRentalKeyModal
            isOpen={editModalOpen}
            onClose={() => {
              setEditModalOpen(false);
              setSelectedKey(null);
            }}
            onUpdated={table.refresh}
            rentalKey={selectedKey}
          />
          <RotateKeyModal
            isOpen={rotateModalOpen}
            onClose={() => {
              setRotateModalOpen(false);
              setSelectedKey(null);
            }}
            onRotated={table.refresh}
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

export default RentalKeyAdminList;
