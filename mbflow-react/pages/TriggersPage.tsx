import React, {useEffect, useState} from 'react';
import {
    Calendar,
    Clock,
    Edit,
    Filter,
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
import {triggerService} from '@/services/triggerService';
import type {Trigger, TriggerStatus, TriggerType,} from '@/types/triggers';
import { Button } from '@/components/ui';

export const TriggersPage: React.FC = () => {
    const [triggers, setTriggers] = useState<Trigger[]>([]);
    const [filteredTriggers, setFilteredTriggers] = useState<Trigger[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [selectedType, setSelectedType] = useState<TriggerType | 'all'>('all');
    const [selectedStatus, setSelectedStatus] = useState<TriggerStatus | 'all'>('all');
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [editingTrigger, setEditingTrigger] = useState<Trigger | null>(null);
    const [deletingId, setDeletingId] = useState<string | null>(null);
    const [togglingId, setTogglingId] = useState<string | null>(null);
    const [executingId, setExecutingId] = useState<string | null>(null);

    useEffect(() => {
        fetchTriggers();
    }, []);

    useEffect(() => {
        applyFilters();
    }, [triggers, selectedType, selectedStatus]);

    const fetchTriggers = async () => {
        setIsLoading(true);
        try {
            const response = await triggerService.getTriggers();
            setTriggers(response.triggers);
        } catch (error) {
            console.error('Failed to fetch triggers:', error);
        } finally {
            setIsLoading(false);
        }
    };

    const applyFilters = () => {
        let filtered = [...triggers];

        if (selectedType !== 'all') {
            filtered = filtered.filter((t) => t.type === selectedType);
        }

        if (selectedStatus !== 'all') {
            filtered = filtered.filter((t) => t.status === selectedStatus);
        }

        setFilteredTriggers(filtered);
    };

    const handleToggleStatus = async (trigger: Trigger) => {
        setTogglingId(trigger.id);
        try {
            const updated =
                trigger.status === 'enabled'
                    ? await triggerService.disableTrigger(trigger.id)
                    : await triggerService.enableTrigger(trigger.id);

            setTriggers((prev) =>
                prev.map((t) => (t.id === trigger.id ? updated : t))
            );
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
            // TODO: Navigate to execution details or show success message
        } catch (error) {
            console.error('Failed to execute trigger:', error);
        } finally {
            setExecutingId(null);
        }
    };

    const handleDelete = async (triggerId: string) => {
        if (!confirm('Are you sure you want to delete this trigger?')) {
            return;
        }

        setDeletingId(triggerId);
        try {
            await triggerService.deleteTrigger(triggerId);
            setTriggers((prev) => prev.filter((t) => t.id !== triggerId));
        } catch (error) {
            console.error('Failed to delete trigger:', error);
        } finally {
            setDeletingId(null);
        }
    };

    const getTriggerIcon = (type: TriggerType) => {
        switch (type) {
            case 'schedule':
                return <Clock size={16}/>;
            case 'webhook':
                return <Webhook size={16}/>;
            case 'event':
                return <Zap size={16}/>;
            case 'manual':
                return <Hand size={16}/>;
            default:
                return <Calendar size={16}/>;
        }
    };

    const getTriggerTypeColor = (type: TriggerType) => {
        switch (type) {
            case 'schedule':
                return 'blue';
            case 'webhook':
                return 'purple';
            case 'event':
                return 'orange';
            case 'manual':
                return 'green';
            default:
                return 'gray';
        }
    };

    const formatNextTrigger = (nextTriggerAt?: string) => {
        if (!nextTriggerAt) return '-';
        const date = new Date(nextTriggerAt);
        const now = new Date();
        const diffMs = date.getTime() - now.getTime();
        const diffMins = Math.floor(diffMs / 60000);

        if (diffMins < 0) return 'Overdue';
        if (diffMins < 60) return `in ${diffMins} mins`;
        const diffHours = Math.floor(diffMins / 60);
        if (diffHours < 24) return `in ${diffHours} hours`;
        const diffDays = Math.floor(diffHours / 24);
        return `in ${diffDays} days`;
    };

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-7xl mx-auto space-y-6">
                {/* Header */}
                <div className="flex justify-between items-end">
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                            Triggers
                        </h1>
                        <p className="text-slate-500 dark:text-slate-400 mt-1">
                            Manage workflow triggers and execution schedules
                        </p>
                    </div>
                    <Button
                        onClick={() => setShowCreateModal(true)}
                        variant="primary"
                        size="sm"
                        icon={<Plus size={16} />}
                    >
                        Create Trigger
                    </Button>
                </div>

                {/* Filters */}
                <div
                    className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 flex flex-wrap gap-4">
                    <div className="flex items-center gap-2">
                        <Filter size={16} className="text-slate-400"/>
                        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
              Filter:
            </span>
                    </div>

                    {/* Type Filter */}
                    <select
                        value={selectedType}
                        onChange={(e) => setSelectedType(e.target.value as TriggerType | 'all')}
                        className="px-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-blue-500 text-slate-700 dark:text-slate-300"
                    >
                        <option value="all">All Types</option>
                        <option value="manual">Manual</option>
                        <option value="schedule">Schedule</option>
                        <option value="webhook">Webhook</option>
                        <option value="event">Event</option>
                    </select>

                    {/* Status Filter */}
                    <select
                        value={selectedStatus}
                        onChange={(e) =>
                            setSelectedStatus(e.target.value as TriggerStatus | 'all')
                        }
                        className="px-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-blue-500 text-slate-700 dark:text-slate-300"
                    >
                        <option value="all">All Status</option>
                        <option value="enabled">Enabled</option>
                        <option value="disabled">Disabled</option>
                    </select>

                    <div className="ml-auto text-sm text-slate-500 dark:text-slate-400">
                        {filteredTriggers.length} of {triggers.length} triggers
                    </div>
                </div>

                {/* Triggers List */}
                {isLoading ? (
                    <div className="flex items-center justify-center h-64">
                        <Loader2 size={32} className="text-blue-500 animate-spin"/>
                    </div>
                ) : filteredTriggers.length === 0 ? (
                    <div
                        className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-12 text-center">
                        <Calendar size={48} className="mx-auto text-slate-300 dark:text-slate-700 mb-4"/>
                        <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300 mb-2">
                            No triggers found
                        </h3>
                        <p className="text-slate-500 dark:text-slate-400 mb-6">
                            {selectedType !== 'all' || selectedStatus !== 'all'
                                ? 'Try adjusting your filters'
                                : 'Create your first trigger to automate workflow execution'}
                        </p>
                        {selectedType === 'all' && selectedStatus === 'all' && (
                            <Button
                                onClick={() => setShowCreateModal(true)}
                                variant="primary"
                                size="sm"
                            >
                                Create Trigger
                            </Button>
                        )}
                    </div>
                ) : (
                    <div
                        className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead
                                    className="text-xs text-slate-500 uppercase bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-800">
                                <tr>
                                    <th className="px-6 py-3 text-left font-medium">Name</th>
                                    <th className="px-6 py-3 text-left font-medium">Type</th>
                                    <th className="px-6 py-3 text-left font-medium">Workflow</th>
                                    <th className="px-6 py-3 text-left font-medium">Status</th>
                                    <th className="px-6 py-3 text-left font-medium">Next Run</th>
                                    <th className="px-6 py-3 text-left font-medium">Last Triggered</th>
                                    <th className="px-6 py-3 text-right font-medium">Actions</th>
                                </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                                {filteredTriggers.map((trigger) => {
                                    const typeColor = getTriggerTypeColor(trigger.type);
                                    const isToggling = togglingId === trigger.id;
                                    const isDeleting = deletingId === trigger.id;
                                    const isExecuting = executingId === trigger.id;

                                    return (
                                        <tr
                                            key={trigger.id}
                                            className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                                        >
                                            <td className="px-6 py-4">
                                                <div>
                                                    <div className="font-medium text-slate-900 dark:text-slate-200">
                                                        {trigger.name}
                                                    </div>
                                                    {trigger.description && (
                                                        <div
                                                            className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                                            {trigger.description}
                                                        </div>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                          <span
                              className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-${typeColor}-50 text-${typeColor}-700 border border-${typeColor}-200 dark:bg-${typeColor}-900/20 dark:text-${typeColor}-400 dark:border-${typeColor}-900/30`}
                          >
                            {getTriggerIcon(trigger.type)}
                              <span className="ml-1.5 capitalize">{trigger.type}</span>
                          </span>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="text-slate-700 dark:text-slate-300">
                                                    {trigger.workflow_name || trigger.workflow_id}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
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
                                                        <Loader2 size={12} className="animate-spin mr-1.5"/>
                                                    ) : trigger.status === 'enabled' ? (
                                                        <Power size={12} className="mr-1.5"/>
                                                    ) : (
                                                        <PowerOff size={12} className="mr-1.5"/>
                                                    )}
                                                    {trigger.status}
                                                </button>
                                            </td>
                                            <td className="px-6 py-4 text-slate-600 dark:text-slate-400">
                                                {trigger.type === 'schedule'
                                                    ? formatNextTrigger(trigger.next_trigger_at)
                                                    : '-'}
                                            </td>
                                            <td className="px-6 py-4 text-slate-500 dark:text-slate-400 text-xs">
                                                {trigger.last_triggered_at
                                                    ? new Date(trigger.last_triggered_at).toLocaleString()
                                                    : 'Never'}
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center justify-end gap-2">
                                                    {trigger.type === 'manual' && (
                                                        <Button
                                                            onClick={() => handleExecute(trigger.id)}
                                                            disabled={isExecuting}
                                                            variant="ghost"
                                                            size="sm"
                                                            icon={isExecuting ? <Loader2 size={16} className="animate-spin"/> : <Play size={16}/>}
                                                            title="Execute Now"
                                                        />
                                                    )}
                                                    <Button
                                                        onClick={() => setEditingTrigger(trigger)}
                                                        variant="ghost"
                                                        size="sm"
                                                        icon={<Edit size={16}/>}
                                                        title="Edit"
                                                    />
                                                    <Button
                                                        onClick={() => handleDelete(trigger.id)}
                                                        disabled={isDeleting}
                                                        variant="ghost"
                                                        size="sm"
                                                        icon={isDeleting ? <Loader2 size={16} className="animate-spin"/> : <Trash2 size={16}/>}
                                                        title="Delete"
                                                        className="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                                                    />
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                })}
                                </tbody>
                            </table>
                        </div>
                    </div>
                )}
            </div>

            {/* Create/Edit Modal (placeholder) */}
            {showCreateModal && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
                    <div
                        className="bg-white dark:bg-slate-900 rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                        <div
                            className="p-6 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center">
                            <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                                Create New Trigger
                            </h2>
                            <Button
                                onClick={() => setShowCreateModal(false)}
                                variant="ghost"
                                size="sm"
                                icon={<X size={20}/>}
                            />
                        </div>
                        <div className="p-6">
                            <p className="text-slate-600 dark:text-slate-400">
                                Trigger creation form will be implemented here.
                            </p>
                            {/* TODO: Implement trigger creation form */}
                        </div>
                    </div>
                </div>
            )}

            {/* Edit Modal (placeholder) */}
            {editingTrigger && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
                    <div
                        className="bg-white dark:bg-slate-900 rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                        <div
                            className="p-6 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center">
                            <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                                Edit Trigger
                            </h2>
                            <Button
                                onClick={() => setEditingTrigger(null)}
                                variant="ghost"
                                size="sm"
                                icon={<X size={20}/>}
                            />
                        </div>
                        <div className="p-6">
                            <p className="text-slate-600 dark:text-slate-400">
                                Editing trigger: {editingTrigger.name}
                            </p>
                            {/* TODO: Implement trigger edit form */}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
