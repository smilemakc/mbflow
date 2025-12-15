import React, { useState, useEffect } from 'react';
import {
  Plus,
  Search,
  Filter,
  Edit,
  Copy,
  Trash2,
  Calendar,
  Clock,
  GitBranch,
  Loader2,
  AlertCircle,
  CheckCircle2,
  FileText,
  Archive,
  BookOpen
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { workflowService } from '@/services/workflowService';
import { WorkflowStatus } from '@/types';
import { WorkflowVariablesGuide } from '@/components/builder/WorkflowVariablesGuide.tsx';
import { toast } from '../lib/toast';
import { Button } from '@/components/ui';

interface Workflow {
  id: string;
  name: string;
  description: string;
  status?: WorkflowStatus;
  nodes: any[];
  edges: any[];
  createdAt: string;
  updatedAt: string;
}

const STATUS_COLORS = {
  draft: {
    bg: 'bg-slate-100 dark:bg-slate-800',
    text: 'text-slate-700 dark:text-slate-300',
    border: 'border-slate-200 dark:border-slate-700',
    icon: FileText
  },
  active: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    text: 'text-green-700 dark:text-green-400',
    border: 'border-green-200 dark:border-green-900/30',
    icon: CheckCircle2
  },
  inactive: {
    bg: 'bg-orange-50 dark:bg-orange-900/20',
    text: 'text-orange-700 dark:text-orange-400',
    border: 'border-orange-200 dark:border-orange-900/30',
    icon: AlertCircle
  },
  archived: {
    bg: 'bg-slate-50 dark:bg-slate-900/20',
    text: 'text-slate-600 dark:text-slate-500',
    border: 'border-slate-200 dark:border-slate-800',
    icon: Archive
  }
};

export const WorkflowsPage: React.FC = () => {
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [filteredWorkflows, setFilteredWorkflows] = useState<Workflow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<WorkflowStatus | 'all'>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const [showVariablesGuide, setShowVariablesGuide] = useState(false);
  const itemsPerPage = 12;

  useEffect(() => {
    loadWorkflows();
  }, []);

  useEffect(() => {
    filterWorkflows();
  }, [workflows, searchQuery, statusFilter]);

  const loadWorkflows = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await workflowService.getAll();
      setWorkflows(data);
    } catch (err) {
      console.error('Failed to load workflows:', err);
      setError('Failed to load workflows. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const filterWorkflows = () => {
    let filtered = [...workflows];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(w =>
        w.name.toLowerCase().includes(query) ||
        (w.description && w.description.toLowerCase().includes(query))
      );
    }

    if (statusFilter !== 'all') {
      filtered = filtered.filter(w => (w.status || 'draft') === statusFilter);
    }

    setFilteredWorkflows(filtered);
  };

  // Reset page to 1 when search/filter changes (separate effect to avoid loops)
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, statusFilter]);

  const handleCreateNew = () => {
    navigate('/builder');
  };

  const handleEdit = (workflowId: string) => {
    navigate(`/builder?id=${workflowId}`);
  };

  const handleClone = async (workflow: Workflow) => {
    try {
      const cloned = await workflowService.create(
        `${workflow.name} (Copy)`,
        workflow.description
      );

      if (workflow.nodes.length > 0 || workflow.edges.length > 0) {
        await workflowService.save({
          id: cloned.id,
          name: cloned.name,
          description: cloned.description,
          nodes: workflow.nodes,
          edges: workflow.edges
        });
      }

      await loadWorkflows();
    } catch (err) {
      console.error('Failed to clone workflow:', err);
      toast.error('Clone Failed', 'Failed to clone workflow. Please try again.');
    }
  };

  const handleDelete = async (workflowId: string, workflowName: string) => {
    if (!confirm(`Are you sure you want to delete "${workflowName}"? This action cannot be undone.`)) {
      return;
    }

    try {
      await workflowService.delete(workflowId);
      await loadWorkflows();
    } catch (err) {
      console.error('Failed to delete workflow:', err);
      toast.error('Delete Failed', 'Failed to delete workflow. Please try again.');
    }
  };

  const formatDate = (dateStr: string): string => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  };

  const totalPages = Math.ceil(filteredWorkflows.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const endIndex = startIndex + itemsPerPage;
  const currentWorkflows = filteredWorkflows.slice(startIndex, endIndex);

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">

        {/* Header */}
        <div className="flex justify-between items-end">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Workflows</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">
              Manage and organize your automation workflows.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Button
              onClick={() => setShowVariablesGuide(!showVariablesGuide)}
              variant={showVariablesGuide ? 'primary' : 'outline'}
              size="sm"
              icon={<BookOpen size={16} />}
            >
              Variables Guide
            </Button>
            <Button
              onClick={handleCreateNew}
              variant="primary"
              size="sm"
              icon={<Plus size={16} />}
            >
              Create New Workflow
            </Button>
          </div>
        </div>

        {/* Variables Guide Modal */}
        {showVariablesGuide && (
          <WorkflowVariablesGuide
            isModal={true}
            onClose={() => setShowVariablesGuide(false)}
          />
        )}

        {/* Filters and Search */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm">
          <div className="flex flex-col md:flex-row gap-4">
            {/* Search */}
            <div className="flex-1 relative">
              <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Search workflows by name or description..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              />
            </div>

            {/* Status Filter */}
            <div className="flex items-center gap-2">
              <Filter size={16} className="text-slate-400" />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as WorkflowStatus | 'all')}
                className="px-4 py-2.5 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              >
                <option value="all">All Status</option>
                <option value="draft">Draft</option>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
                <option value="archived">Archived</option>
              </select>
            </div>
          </div>

          {/* Results Count */}
          <div className="mt-3 pt-3 border-t border-slate-100 dark:border-slate-800">
            <p className="text-sm text-slate-600 dark:text-slate-400">
              Showing <span className="font-medium text-slate-900 dark:text-white">{filteredWorkflows.length}</span> workflow{filteredWorkflows.length !== 1 ? 's' : ''}
              {searchQuery && <> matching "<span className="font-medium">{searchQuery}</span>"</>}
            </p>
          </div>
        </div>

        {/* Loading State */}
        {isLoading && (
          <div className="flex items-center justify-center py-20">
            <Loader2 size={32} className="animate-spin text-blue-600" />
          </div>
        )}

        {/* Error State */}
        {error && !isLoading && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-900/30 rounded-xl p-6 text-center">
            <AlertCircle size={32} className="mx-auto mb-3 text-red-600 dark:text-red-400" />
            <p className="text-red-800 dark:text-red-400 font-medium">{error}</p>
            <div className="mt-4">
              <Button
                onClick={loadWorkflows}
                variant="danger"
                size="sm"
              >
                Try Again
              </Button>
            </div>
          </div>
        )}

        {/* Empty State */}
        {!isLoading && !error && filteredWorkflows.length === 0 && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-12 text-center">
            <GitBranch size={48} className="mx-auto mb-4 text-slate-300 dark:text-slate-700" />
            <h3 className="text-lg font-bold text-slate-900 dark:text-white mb-2">
              {searchQuery || statusFilter !== 'all' ? 'No workflows found' : 'No workflows yet'}
            </h3>
            <p className="text-slate-500 dark:text-slate-400 mb-6">
              {searchQuery || statusFilter !== 'all'
                ? 'Try adjusting your search or filter criteria.'
                : 'Get started by creating your first workflow.'}
            </p>
            {!searchQuery && statusFilter === 'all' && (
              <Button
                onClick={handleCreateNew}
                variant="primary"
                size="sm"
                icon={<Plus size={16} />}
              >
                Create Your First Workflow
              </Button>
            )}
          </div>
        )}

        {/* Workflow Cards Grid */}
        {!isLoading && !error && currentWorkflows.length > 0 && (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {currentWorkflows.map((workflow) => {
                const status = (workflow.status || 'draft') as WorkflowStatus;
                const statusConfig = STATUS_COLORS[status];
                const StatusIcon = statusConfig.icon;

                return (
                  <div
                    key={workflow.id}
                    className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm hover:shadow-md transition-all group"
                  >
                    {/* Card Header */}
                    <div className="p-5 border-b border-slate-100 dark:border-slate-800">
                      <div className="flex items-start justify-between mb-3">
                        <h3 className="text-lg font-bold text-slate-900 dark:text-white line-clamp-1 flex-1 pr-2">
                          {workflow.name}
                        </h3>
                        <span className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium border ${statusConfig.bg} ${statusConfig.text} ${statusConfig.border} shrink-0`}>
                          <StatusIcon size={12} className="mr-1" />
                          {status.charAt(0).toUpperCase() + status.slice(1)}
                        </span>
                      </div>

                      {workflow.description && (
                        <p className="text-sm text-slate-600 dark:text-slate-400 line-clamp-2">
                          {workflow.description}
                        </p>
                      )}
                    </div>

                    {/* Card Body */}
                    <div className="p-5 space-y-3">
                      {/* Stats */}
                      <div className="flex items-center gap-4 text-sm">
                        <div className="flex items-center text-slate-600 dark:text-slate-400">
                          <GitBranch size={14} className="mr-1.5" />
                          <span className="font-medium">{workflow.nodes?.length || 0}</span>
                          <span className="ml-1">node{workflow.nodes?.length !== 1 ? 's' : ''}</span>
                        </div>
                      </div>

                      {/* Dates */}
                      <div className="space-y-1.5 text-xs">
                        <div className="flex items-center text-slate-500 dark:text-slate-500">
                          <Calendar size={12} className="mr-1.5" />
                          <span>Created {formatDate(workflow.createdAt)}</span>
                        </div>
                        <div className="flex items-center text-slate-500 dark:text-slate-500">
                          <Clock size={12} className="mr-1.5" />
                          <span>Updated {formatDate(workflow.updatedAt)}</span>
                        </div>
                      </div>
                    </div>

                    {/* Card Actions */}
                    <div className="p-4 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-100 dark:border-slate-800 flex items-center gap-2">
                      <Button
                        onClick={() => handleEdit(workflow.id)}
                        variant="primary"
                        size="sm"
                        icon={<Edit size={14} />}
                        className="flex-1"
                      >
                        Edit
                      </Button>
                      <Button
                        onClick={() => handleClone(workflow)}
                        variant="outline"
                        size="sm"
                        icon={<Copy size={14} />}
                        title="Clone workflow"
                      />
                      <Button
                        onClick={() => handleDelete(workflow.id, workflow.name)}
                        variant="danger"
                        size="sm"
                        icon={<Trash2 size={14} />}
                        title="Delete workflow"
                      />
                    </div>
                  </div>
                );
              })}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 pt-4">
                <Button
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  variant="outline"
                  size="sm"
                >
                  Previous
                </Button>

                <div className="flex items-center gap-1">
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
                    <Button
                      key={page}
                      onClick={() => setCurrentPage(page)}
                      variant={currentPage === page ? 'primary' : 'outline'}
                      size="sm"
                    >
                      {page}
                    </Button>
                  ))}
                </div>

                <Button
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                  variant="outline"
                  size="sm"
                >
                  Next
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
