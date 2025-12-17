import React, { useState } from 'react';
import {
  Calendar,
  Clock,
  Edit,
  Hand,
  Loader2,
  Play,
  Plus,
  Power,
  PowerOff,
  Trash2,
  Webhook,
  X,
  Zap,
} from 'lucide-react';
import { triggerService } from '@/services/triggerService';
import type { Trigger, TriggerStatus, TriggerType } from '@/types/triggers';
import { Button, ConfirmModal } from '@/components/ui';
import { DataTable, Column } from '@/components/ui/DataTable';
import { FilterBar, FilterSelect, FilterResults } from '@/components/ui/FilterBar';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { useTableData } from '@/hooks/useTableData';
import { useTranslation } from '../store/translations';

interface TriggerFilters {
  type?: TriggerType;
  status?: TriggerStatus;
}

export const TriggersPage: React.FC = () => {
  const t = useTranslation();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingTrigger, setEditingTrigger] = useState<Trigger | null>(null);
  const [triggerToDelete, setTriggerToDelete] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [executingId, setExecutingId] = useState<string | null>(null);

  const table = useTableData<Trigger, TriggerFilters>({
    fetchFn: async ({ filters }) => {
      const response = await triggerService.getTriggers();
      let filtered = response.triggers;

      if (filters?.type) {
        filtered = filtered.filter((t) => t.type === filters.type);
      }

      if (filters?.status) {
        filtered = filtered.filter((t) => t.status === filters.status);
      }

      return {
        items: filtered,
        total: filtered.length,
      };
    },
    initialFilters: {},
  });

  const handleToggleStatus = async (trigger: Trigger) => {
    setTogglingId(trigger.id);
    try {
      const updated =
        trigger.status === 'enabled'
          ? await triggerService.disableTrigger(trigger.id)
          : await triggerService.enableTrigger(trigger.id);

      table.refresh();
    } catch (error) {
      console.error('Failed to toggle trigger status:', error);
    } finally {
      setTogglingId(null);
    }
  };

  const handleExecute = async (triggerId: string) => {
    setExecutingId(triggerId);
    try {
      const result = await triggerService.executeTrigger(triggerId);
      console.log('Trigger executed, execution ID:', result.execution_id);
    } catch (error) {
      console.error('Failed to execute trigger:', error);
    } finally {
      setExecutingId(null);
    }
  };

  const handleDelete = async () => {
    if (!triggerToDelete) return;

    try {
      await triggerService.deleteTrigger(triggerToDelete);
      table.refresh();
    } catch (error) {
      console.error('Failed to delete trigger:', error);
    } finally {
      setTriggerToDelete(null);
    }
  };

  const getTriggerIcon = (type: TriggerType) => {
    switch (type) {
      case 'schedule':
        return <Clock size={16} />;
      case 'webhook':
        return <Webhook size={16} />;
      case 'event':
        return <Zap size={16} />;
      case 'manual':
        return <Hand size={16} />;
      default:
        return <Calendar size={16} />;
    }
  };

  const getTriggerTypeVariant = (type: TriggerType) => {
    switch (type) {
      case 'schedule':
        return 'info';
      case 'webhook':
        return { bg: 'bg-purple-50 dark:bg-purple-900/20', text: 'text-purple-700 dark:text-purple-400', border: 'border-purple-200 dark:border-purple-900/30' };
      case 'event':
        return 'warning';
      case 'manual':
        return 'success';
      default:
        return 'neutral';
    }
  };

  const formatNextTrigger = (nextTriggerAt?: string) => {
    if (!nextTriggerAt) return '-';
    const date = new Date(nextTriggerAt);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 0) return t.triggers.overdue;
    if (diffMins < 60) return t.triggers.inMinutes.replace('{n}', diffMins.toString());
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return t.triggers.inHours.replace('{n}', diffHours.toString());
    const diffDays = Math.floor(diffHours / 24);
    return t.triggers.inDays.replace('{n}', diffDays.toString());
  };

  const columns: Column<Trigger>[] = [
    {
      key: 'name',
      header: t.triggers.table.name,
      render: (trigger) => (
        <div>
          <div className="font-medium text-slate-900 dark:text-slate-200">{trigger.name}</div>
          {trigger.description && (
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{trigger.description}</div>
          )}
        </div>
      ),
    },
    {
      key: 'type',
      header: t.triggers.table.type,
      render: (trigger) => (
        <StatusBadge
          status={trigger.type}
          variant={getTriggerTypeVariant(trigger.type)}
          icon={getTriggerIcon(trigger.type)}
        />
      ),
    },
    {
      key: 'workflow',
      header: t.triggers.table.workflow,
      render: (trigger) => (
        <div className="text-slate-700 dark:text-slate-300">
          {trigger.workflow_name || trigger.workflow_id}
        </div>
      ),
    },
    {
      key: 'status',
      header: t.triggers.table.status,
      render: (trigger) => {
        const isToggling = togglingId === trigger.id;
        return (
          <button
            onClick={() => handleToggleStatus(trigger)}
            disabled={isToggling}
            className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border transition-colors ${
              trigger.status === 'enabled'
                ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30 hover:bg-green-100 dark:hover:bg-green-900/30'
                : 'bg-slate-50 text-slate-700 border-slate-200 dark:bg-slate-800/50 dark:text-slate-400 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800'
            } ${isToggling ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
          >
            {isToggling ? (
              <Loader2 size={12} className="animate-spin mr-1.5" />
            ) : trigger.status === 'enabled' ? (
              <Power size={12} className="mr-1.5" />
            ) : (
              <PowerOff size={12} className="mr-1.5" />
            )}
            {trigger.status}
          </button>
        );
      },
    },
    {
      key: 'next_trigger_at',
      header: t.triggers.table.nextRun,
      render: (trigger) => (
        <span className="text-slate-600 dark:text-slate-400">
          {trigger.type === 'schedule' ? formatNextTrigger(trigger.next_trigger_at) : '-'}
        </span>
      ),
    },
    {
      key: 'last_triggered_at',
      header: t.triggers.table.lastTriggered,
      render: (trigger) => (
        <span className="text-slate-500 dark:text-slate-400 text-xs">
          {trigger.last_triggered_at
            ? new Date(trigger.last_triggered_at).toLocaleString()
            : t.triggers.never}
        </span>
      ),
    },
  ];

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex justify-between items-end">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t.triggers.title}</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">{t.triggers.subtitle}</p>
          </div>
          <Button
            onClick={() => setShowCreateModal(true)}
            variant="primary"
            size="sm"
            icon={<Plus size={16} />}
          >
            {t.triggers.createTrigger}
          </Button>
        </div>

        <FilterBar>
          <FilterSelect
            label={t.triggers.filter}
            value={(table.filters.type as string) || 'all'}
            onChange={(value) =>
              table.setFilters({
                ...table.filters,
                type: value === 'all' ? undefined : (value as TriggerType),
              })
            }
            options={[
              { label: t.triggers.allTypes, value: 'all' },
              { label: t.triggers.manual, value: 'manual' },
              { label: t.triggers.schedule, value: 'schedule' },
              { label: t.triggers.webhook, value: 'webhook' },
              { label: t.triggers.event, value: 'event' },
            ]}
          />
          <FilterSelect
            value={(table.filters.status as string) || 'all'}
            onChange={(value) =>
              table.setFilters({
                ...table.filters,
                status: value === 'all' ? undefined : (value as TriggerStatus),
              })
            }
            options={[
              { label: t.triggers.allStatus, value: 'all' },
              { label: t.triggers.enabled, value: 'enabled' },
              { label: t.triggers.disabled, value: 'disabled' },
            ]}
          />
          <FilterResults total={table.total} itemLabel={t.triggers.triggersCount} />
        </FilterBar>

        <DataTable
          data={table.items}
          columns={columns}
          keyExtractor={(trigger) => trigger.id}
          loading={table.loading}
          error={table.error}
          emptyIcon={Calendar}
          emptyTitle={t.triggers.noTriggersFound}
          emptyDescription={
            table.filters.type || table.filters.status
              ? t.triggers.adjustFilters
              : t.triggers.createFirst
          }
          emptyAction={
            !table.filters.type && !table.filters.status
              ? {
                  label: t.triggers.createTrigger,
                  onClick: () => setShowCreateModal(true),
                }
              : undefined
          }
          actions={(trigger) => (
            <>
              {trigger.type === 'manual' && (
                <Button
                  onClick={() => handleExecute(trigger.id)}
                  disabled={executingId === trigger.id}
                  variant="ghost"
                  size="sm"
                  icon={
                    executingId === trigger.id ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : (
                      <Play size={16} />
                    )
                  }
                  title={t.triggers.executeNow}
                />
              )}
              <Button
                onClick={() => setEditingTrigger(trigger)}
                variant="ghost"
                size="sm"
                icon={<Edit size={16} />}
                title={t.triggers.edit}
              />
              <Button
                onClick={() => setTriggerToDelete(trigger.id)}
                variant="ghost"
                size="sm"
                icon={<Trash2 size={16} />}
                title={t.common.delete}
                className="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
              />
            </>
          )}
        />
      </div>

      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-slate-900 rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center">
              <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                {t.triggers.createModal.title}
              </h2>
              <Button
                onClick={() => setShowCreateModal(false)}
                variant="ghost"
                size="sm"
                icon={<X size={20} />}
              />
            </div>
            <div className="p-6">
              <p className="text-slate-600 dark:text-slate-400">
                {t.triggers.createModal.placeholder}
              </p>
            </div>
          </div>
        </div>
      )}

      {editingTrigger && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-slate-900 rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center">
              <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                {t.triggers.editModal.title}
              </h2>
              <Button
                onClick={() => setEditingTrigger(null)}
                variant="ghost"
                size="sm"
                icon={<X size={20} />}
              />
            </div>
            <div className="p-6">
              <p className="text-slate-600 dark:text-slate-400">
                {t.triggers.editModal.editing} {editingTrigger.name}
              </p>
            </div>
          </div>
        </div>
      )}

      <ConfirmModal
        isOpen={!!triggerToDelete}
        onClose={() => setTriggerToDelete(null)}
        onConfirm={handleDelete}
        title={t.triggers.deleteModal.title}
        message={t.triggers.deleteModal.message}
        confirmText={t.triggers.deleteModal.confirm}
        variant="danger"
      />
    </div>
  );
};
