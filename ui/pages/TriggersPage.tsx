import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  Repeat,
  Trash2,
  Webhook,
  Zap,
} from 'lucide-react';
import { triggerService } from '@/services/triggerService';
import { workflowService } from '@/services/workflowService';
import type { Trigger, TriggerType } from '@/types/triggers';
import { TRIGGER_TYPES } from '@/types/triggers';
import { Button, ConfirmModal } from '@/components/ui';
import { DataTable, Column } from '@/components/ui/DataTable';
import { FilterBar, FilterSelect, FilterResults } from '@/components/ui/FilterBar';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { TriggerFormModal, type WorkflowOption } from '@/components/triggers/TriggerFormModal';
import { useTableData } from '@/hooks/useTableData';
import { getErrorMessage } from '@/lib/api';
import { toast } from '@/lib/toast';
import {
  filterTriggersByStatus,
  getTriggerDescription,
  getTriggerName,
  relativeTime,
  summarizeTriggerConfig,
  type TriggerStatusFilter,
} from '@/lib/triggers';
import { useTranslation } from '../store/translations';

interface TriggerFilters {
  type?: TriggerType;
  status?: TriggerStatusFilter;
}

const TYPE_ICONS: Record<TriggerType, React.ReactNode> = {
  cron: <Clock size={16} />,
  interval: <Repeat size={16} />,
  webhook: <Webhook size={16} />,
  event: <Zap size={16} />,
  manual: <Hand size={16} />,
};

const TYPE_VARIANTS: Record<TriggerType, React.ComponentProps<typeof StatusBadge>['variant']> = {
  cron: 'info',
  interval: 'info',
  webhook: {
    bg: 'bg-purple-50 dark:bg-purple-900/20',
    text: 'text-purple-700 dark:text-purple-400',
    border: 'border-purple-200 dark:border-purple-900/30',
  },
  event: 'warning',
  manual: 'success',
};

export const TriggersPage: React.FC = () => {
  const t = useTranslation();
  const [formOpen, setFormOpen] = useState(false);
  const [editingTrigger, setEditingTrigger] = useState<Trigger | null>(null);
  const [triggerToDelete, setTriggerToDelete] = useState<Trigger | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [executingId, setExecutingId] = useState<string | null>(null);
  const [workflows, setWorkflows] = useState<WorkflowOption[]>([]);
  const [workflowsLoading, setWorkflowsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    workflowService
      .getAll()
      .then((list) => {
        if (!cancelled) setWorkflows(list.map((wf) => ({ id: wf.id, name: wf.name })));
      })
      .catch(() => {
        // The list still works without names; ids are shown instead.
      })
      .finally(() => {
        if (!cancelled) setWorkflowsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const workflowNames = useMemo(() => new Map(workflows.map((wf) => [wf.id, wf.name])), [workflows]);

  const table = useTableData<Trigger, TriggerFilters>({
    fetchFn: async ({ filters }) => {
      // Type filters server-side; the API has no enabled filter, so status is applied here.
      const triggers = await triggerService.getAllTriggers({ type: filters?.type });
      const items = filterTriggersByStatus(triggers, filters?.status);
      return { items, total: items.length };
    },
    initialFilters: {},
  });

  const typeLabels: Record<TriggerType, string> = {
    manual: t.triggers.manual,
    cron: t.triggers.cron,
    interval: t.triggers.interval,
    webhook: t.triggers.webhook,
    event: t.triggers.event,
  };

  const formatRelative = useCallback(
    (date: string) => {
      const rel = relativeTime(date);
      if (rel.unit === 'now') return t.triggers.justNow;
      const n = rel.value.toString();
      if (rel.past) {
        const key = { minutes: 'minutesAgo', hours: 'hoursAgo', days: 'daysAgo' } as const;
        return t.triggers[key[rel.unit]].replace('{n}', n);
      }
      const key = { minutes: 'inMinutes', hours: 'inHours', days: 'inDays' } as const;
      return t.triggers[key[rel.unit]].replace('{n}', n);
    },
    [t],
  );

  const openCreate = () => {
    setEditingTrigger(null);
    setFormOpen(true);
  };

  const openEdit = (trigger: Trigger) => {
    setEditingTrigger(trigger);
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setEditingTrigger(null);
  };

  const handleSaved = (_trigger: Trigger, mode: 'create' | 'edit') => {
    toast.success(mode === 'create' ? t.triggers.created : t.triggers.updated);
    closeForm();
    table.refresh();
  };

  const handleToggleStatus = async (trigger: Trigger) => {
    setTogglingId(trigger.id);
    try {
      if (trigger.enabled) {
        await triggerService.disableTrigger(trigger.id);
      } else {
        await triggerService.enableTrigger(trigger.id);
      }
      table.refresh();
    } catch (error) {
      toast.error(t.triggers.enableFailed, getErrorMessage(error));
    } finally {
      setTogglingId(null);
    }
  };

  const handleExecute = async (trigger: Trigger) => {
    setExecutingId(trigger.id);
    try {
      const result = await triggerService.executeTrigger(trigger.id);
      toast.success(
        t.triggers.executeStarted,
        result?.execution_id ? t.triggers.executeStartedWithId.replace('{id}', result.execution_id) : undefined,
      );
      table.refresh();
    } catch (error) {
      toast.error(t.triggers.executeFailed, getErrorMessage(error));
    } finally {
      setExecutingId(null);
    }
  };

  const handleDelete = async () => {
    if (!triggerToDelete) return;
    const target = triggerToDelete;
    setTriggerToDelete(null);
    try {
      await triggerService.deleteTrigger(target.id);
      toast.success(t.triggers.deleted);
      table.refresh();
    } catch (error) {
      toast.error(t.triggers.deleteFailed, getErrorMessage(error));
    }
  };

  const columns: Column<Trigger>[] = [
    {
      key: 'name',
      header: t.triggers.table.name,
      render: (trigger) => {
        const description = getTriggerDescription(trigger);
        return (
          <div>
            <div className="font-medium text-slate-900 dark:text-slate-200">{getTriggerName(trigger)}</div>
            {description && <div className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{description}</div>}
          </div>
        );
      },
    },
    {
      key: 'type',
      header: t.triggers.table.type,
      render: (trigger) => (
        <StatusBadge
          status={typeLabels[trigger.type] ?? trigger.type}
          variant={TYPE_VARIANTS[trigger.type] ?? 'neutral'}
          icon={TYPE_ICONS[trigger.type] ?? <Calendar size={16} />}
        />
      ),
    },
    {
      key: 'workflow',
      header: t.triggers.table.workflow,
      render: (trigger) => (
        <div className="text-slate-700 dark:text-slate-300">
          {workflowNames.get(trigger.workflow_id) || (
            <span className="font-mono text-xs">{trigger.workflow_id}</span>
          )}
        </div>
      ),
    },
    {
      key: 'config',
      header: t.triggers.table.config,
      render: (trigger) => {
        const summary = summarizeTriggerConfig(trigger);
        return (
          <div className="text-xs">
            {summary ? (
              <span className="font-mono text-slate-600 dark:text-slate-400">{summary}</span>
            ) : (
              <span className="text-slate-400">-</span>
            )}
            {trigger.next_run && (
              <div className="text-slate-500 dark:text-slate-400 mt-0.5">
                {t.triggers.table.nextRun}: {formatRelative(trigger.next_run)}
              </div>
            )}
          </div>
        );
      },
    },
    {
      key: 'status',
      header: t.triggers.table.status,
      render: (trigger) => {
        const isToggling = togglingId === trigger.id;
        return (
          <button
            type="button"
            onClick={() => handleToggleStatus(trigger)}
            disabled={isToggling}
            title={trigger.enabled ? t.triggers.disable : t.triggers.enable}
            aria-pressed={trigger.enabled}
            className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border transition-colors ${
              trigger.enabled
                ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30 hover:bg-green-100 dark:hover:bg-green-900/30'
                : 'bg-slate-50 text-slate-700 border-slate-200 dark:bg-slate-800/50 dark:text-slate-400 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800'
            } ${isToggling ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
          >
            {isToggling ? (
              <Loader2 size={12} className="animate-spin mr-1.5" />
            ) : trigger.enabled ? (
              <Power size={12} className="mr-1.5" />
            ) : (
              <PowerOff size={12} className="mr-1.5" />
            )}
            {trigger.enabled ? t.triggers.enabled : t.triggers.disabled}
          </button>
        );
      },
    },
    {
      key: 'last_run',
      header: t.triggers.table.lastTriggered,
      render: (trigger) => (
        <span
          className="text-slate-500 dark:text-slate-400 text-xs"
          title={trigger.last_run ? new Date(trigger.last_run).toLocaleString() : undefined}
        >
          {trigger.last_run ? formatRelative(trigger.last_run) : t.triggers.never}
        </span>
      ),
    },
  ];

  const hasFilters = !!(table.filters.type || table.filters.status);

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex justify-between items-end">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t.triggers.title}</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">{t.triggers.subtitle}</p>
          </div>
          <Button onClick={openCreate} variant="primary" size="sm" icon={<Plus size={16} />}>
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
              ...TRIGGER_TYPES.map((type) => ({ label: typeLabels[type], value: type })),
            ]}
          />
          <FilterSelect
            value={(table.filters.status as string) || 'all'}
            onChange={(value) =>
              table.setFilters({
                ...table.filters,
                status: value === 'all' ? undefined : (value as TriggerStatusFilter),
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
          emptyDescription={hasFilters ? t.triggers.adjustFilters : t.triggers.createFirst}
          emptyAction={hasFilters ? undefined : { label: t.triggers.createTrigger, onClick: openCreate }}
          actions={(trigger) => (
            <>
              <Button
                onClick={() => handleExecute(trigger)}
                disabled={executingId === trigger.id || !trigger.enabled}
                variant="ghost"
                size="sm"
                icon={
                  executingId === trigger.id ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />
                }
                title={t.triggers.executeNow}
                aria-label={t.triggers.executeNow}
              />
              <Button
                onClick={() => openEdit(trigger)}
                variant="ghost"
                size="sm"
                icon={<Edit size={16} />}
                title={t.triggers.edit}
                aria-label={t.triggers.edit}
              />
              <Button
                onClick={() => setTriggerToDelete(trigger)}
                variant="ghost"
                size="sm"
                icon={<Trash2 size={16} />}
                title={t.common.delete}
                aria-label={t.common.delete}
                className="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
              />
            </>
          )}
        />
      </div>

      <TriggerFormModal
        isOpen={formOpen}
        trigger={editingTrigger}
        workflows={workflows}
        workflowsLoading={workflowsLoading}
        onClose={closeForm}
        onSaved={handleSaved}
      />

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
