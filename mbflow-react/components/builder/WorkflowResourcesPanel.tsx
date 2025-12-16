import React, { useState, useEffect } from 'react';
import { X, Plus, Copy, Trash2, Database, Check } from 'lucide-react';
import { useDagStore } from '@/store/dagStore';
import { resourcesApi, FileStorageResource } from '@/services/resources';
import { WorkflowResource } from '@/types/workflow';
import { Button } from '@/components/ui';

interface WorkflowResourcesPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

export const WorkflowResourcesPanel: React.FC<WorkflowResourcesPanelProps> = ({ isOpen, onClose }) => {
  const { resources, attachResource, detachResource, updateResourceAlias } = useDagStore();
  const [availableResources, setAvailableResources] = useState<FileStorageResource[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedResourceId, setSelectedResourceId] = useState('');
  const [newAlias, setNewAlias] = useState('');
  const [editingAlias, setEditingAlias] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [copiedAlias, setCopiedAlias] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      loadAvailableResources();
    }
  }, [isOpen]);

  const loadAvailableResources = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await resourcesApi.listResources();
      setAvailableResources(response.data.resources || []);
    } catch (error) {
      console.error('Failed to load resources:', error);
      setError('Failed to load resources. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const unattachedResources = availableResources.filter(
    r => !resources.some(wr => wr.resource_id === r.id)
  );

  const handleAttach = async () => {
    if (!selectedResourceId || !newAlias.trim()) return;

    setError(null);
    try {
      await attachResource(selectedResourceId, newAlias.trim());
      setSelectedResourceId('');
      setNewAlias('');
    } catch (error) {
      console.error('Failed to attach resource:', error);
      setError('Failed to attach resource. Please try again.');
    }
  };

  const handleDetach = async (resourceId: string) => {
    setError(null);
    try {
      await detachResource(resourceId);
    } catch (error) {
      console.error('Failed to detach resource:', error);
      setError('Failed to detach resource. Please try again.');
    }
  };

  const handleStartEdit = (resource: WorkflowResource) => {
    setEditingAlias(resource.resource_id);
    setEditValue(resource.alias);
  };

  const handleSaveEdit = async (resourceId: string) => {
    if (editValue.trim() && editValue !== resources.find(r => r.resource_id === resourceId)?.alias) {
      setError(null);
      try {
        await updateResourceAlias(resourceId, editValue.trim());
        setEditingAlias(null);
        setEditValue('');
      } catch (error) {
        console.error('Failed to update alias:', error);
        setError('Failed to update alias. Please try again.');
      }
    } else {
      setEditingAlias(null);
      setEditValue('');
    }
  };

  const handleCopyTemplate = (alias: string) => {
    const template = `{{resource.${alias}}}`;
    navigator.clipboard.writeText(template);
    setCopiedAlias(alias);
    setTimeout(() => setCopiedAlias(null), 2000);
  };

  const getResourceName = (resourceId: string) => {
    return availableResources.find(r => r.id === resourceId)?.name || resourceId;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-2xl max-h-[80vh] bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center bg-slate-50 dark:bg-slate-800/50">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
              <Database size={20} className="text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <h2 className="font-bold text-slate-800 dark:text-slate-100">
                Workflow Resources
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Manage resources attached to this workflow
              </p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            icon={<X size={18} />}
            onClick={onClose}
          />
        </div>

        {/* Info Banner */}
        <div className="mx-4 mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/30 rounded-lg">
          <p className="text-sm text-blue-700 dark:text-blue-300">
            Attach resources to use them in node configurations via <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-800 rounded font-mono text-xs">{'{{resource.alias}}'}</code> syntax.
          </p>
        </div>

        {/* Error Banner */}
        {error && (
          <div className="mx-4 mt-2 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-900/30 rounded-lg flex items-start gap-2">
            <X size={16} className="text-red-600 dark:text-red-400 mt-0.5 shrink-0" />
            <div className="flex-1">
              <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            </div>
            <button
              onClick={() => setError(null)}
              className="text-red-400 hover:text-red-600 dark:hover:text-red-200"
            >
              <X size={14} />
            </button>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {/* Add Resource Form */}
          <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg space-y-3">
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">Add Resource</h3>
            <div className="flex gap-2">
              <select
                value={selectedResourceId}
                onChange={(e) => setSelectedResourceId(e.target.value)}
                className="flex-1 px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
                disabled={loading}
              >
                <option value="">Select a resource...</option>
                {unattachedResources.map(r => (
                  <option key={r.id} value={r.id}>
                    {r.name} ({r.type})
                  </option>
                ))}
              </select>
              <input
                type="text"
                value={newAlias}
                onChange={(e) => setNewAlias(e.target.value)}
                placeholder="Alias (e.g., mainStorage)"
                className="w-48 px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
              />
              <Button
                variant="primary"
                size="sm"
                icon={<Plus size={16} />}
                onClick={handleAttach}
                disabled={!selectedResourceId || !newAlias.trim()}
              >
                Add
              </Button>
            </div>
          </div>

          {/* Attached Resources List */}
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">
              Attached Resources ({resources.length})
            </h3>

            {resources.length === 0 ? (
              <div className="text-center py-8 text-slate-400">
                <Database size={32} className="mx-auto mb-2 opacity-50" />
                <p className="text-sm">No resources attached yet</p>
                <p className="text-xs mt-1">Add a resource above to use it in your workflow</p>
              </div>
            ) : (
              <div className="space-y-2">
                {resources.map(resource => (
                  <div
                    key={resource.resource_id}
                    className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg group"
                  >
                    <Database size={16} className="text-slate-400 shrink-0" />
                    <div className="flex-1 min-w-0">
                      {editingAlias === resource.resource_id ? (
                        <input
                          type="text"
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          onBlur={() => handleSaveEdit(resource.resource_id)}
                          onKeyDown={(e) => e.key === 'Enter' && handleSaveEdit(resource.resource_id)}
                          className="px-2 py-1 text-sm font-mono bg-white dark:bg-slate-900 border border-blue-500 rounded w-full focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                          autoFocus
                        />
                      ) : (
                        <div>
                          <span
                            className="font-mono text-sm text-slate-900 dark:text-white cursor-pointer hover:text-blue-500"
                            onClick={() => handleStartEdit(resource)}
                          >
                            {resource.alias}
                          </span>
                          <span className="ml-2 text-xs text-slate-500">
                            ({getResourceName(resource.resource_id)})
                          </span>
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        variant="ghost"
                        size="sm"
                        icon={copiedAlias === resource.alias ? <Check size={14} className="text-green-500" /> : <Copy size={14} />}
                        onClick={() => handleCopyTemplate(resource.alias)}
                        title="Copy template"
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        icon={<Trash2 size={14} />}
                        onClick={() => handleDetach(resource.resource_id)}
                        title="Remove resource"
                        className="text-red-500 hover:text-red-600 hover:bg-red-100 dark:hover:bg-red-900/30"
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-100 dark:border-slate-800 flex justify-end bg-slate-50 dark:bg-slate-800/50">
          <Button
            variant="outline"
            onClick={onClose}
          >
            Close
          </Button>
        </div>
      </div>
    </div>
  );
};

export default WorkflowResourcesPanel;
