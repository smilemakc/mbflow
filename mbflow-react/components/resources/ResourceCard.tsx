/**
 * ResourceCard component
 * Single Responsibility: Displays a single file storage resource with its details and actions
 */

import React from 'react';
import {AlertCircle, Calendar, CheckCircle2, Database, Eye, FileText, Trash2,} from 'lucide-react';
import {Button} from '@/components/ui';
import {FileStorageResource} from '@/services/resources.ts';
import {formatBytes, formatShortDate} from '@/utils/formatters.ts';
import {useTranslation} from '@/store/translations';

interface ResourceCardProps {
    resource: FileStorageResource;
    onDelete: (id: string, name: string) => void;
    onViewFiles: (resource: FileStorageResource) => void;
}

export const ResourceCard: React.FC<ResourceCardProps> = (
    {
        resource,
        onDelete,
        onViewFiles,
    }) => {
    const t = useTranslation();
    const isActive = resource.status === 'active';

    return (
        <div
            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm hover:shadow-md transition-all group">
            <ResourceHeader resource={resource} isActive={isActive}/>
            <StorageUsage resource={resource}/>
            <ResourceActions
                resource={resource}
                onDelete={onDelete}
                onViewFiles={onViewFiles}
            />
        </div>
    );
};

interface ResourceHeaderProps {
    resource: FileStorageResource;
    isActive: boolean;
}

const ResourceHeader: React.FC<ResourceHeaderProps> = ({resource, isActive}) => (
    <div className="flex justify-between items-start mb-4">
        <div className="flex items-start space-x-3 flex-1">
            <div
                className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-100 dark:border-blue-900/30">
                <Database size={20} className="text-blue-600 dark:text-blue-400"/>
            </div>
            <div className="flex-1 min-w-0">
                <h3 className="font-bold text-slate-900 dark:text-white truncate">
                    {resource.name}
                </h3>
                {resource.description && (
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-2">
                        {resource.description}
                    </p>
                )}
            </div>
        </div>
        <StatusBadge status={resource.status} isActive={isActive}/>
    </div>
);

interface StatusBadgeProps {
    status: string;
    isActive: boolean;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({status, isActive}) => (
    <span
        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium shrink-0 ml-2 ${
            isActive
                ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400'
                : 'bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-400'
        }`}
    >
    {isActive ? (
        <CheckCircle2 size={10} className="mr-1"/>
    ) : (
        <AlertCircle size={10} className="mr-1"/>
    )}
        {status}
  </span>
);

interface StorageUsageProps {
    resource: FileStorageResource;
}

const StorageUsage: React.FC<StorageUsageProps> = ({resource}) => {
    const t = useTranslation();
    const getProgressColor = (percent: number): string => {
        if (percent > 90) return 'bg-red-600';
        if (percent > 75) return 'bg-yellow-500';
        return 'bg-blue-600';
    };

    const getTextColor = (percent: number): string => {
        if (percent > 90) return 'text-red-600 dark:text-red-400';
        if (percent > 75) return 'text-yellow-600 dark:text-yellow-400';
        return 'text-slate-600 dark:text-slate-400';
    };

    return (
        <div className="space-y-3">
            <div>
                <div className="flex justify-between text-xs mb-1.5">
                    <span className="text-slate-600 dark:text-slate-400">{t.resources.storageUsage}</span>
                    <span className="font-medium text-slate-900 dark:text-white">
            {formatBytes(resource.used_storage_bytes)} / {formatBytes(resource.storage_limit_bytes)}
          </span>
                </div>
                <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2 overflow-hidden">
                    <div
                        className={`h-2 rounded-full transition-all ${getProgressColor(resource.usage_percent)}`}
                        style={{width: `${Math.min(resource.usage_percent, 100)}%`}}
                    />
                </div>
                <div className="flex items-center justify-between mt-1.5 text-xs">
          <span className="text-slate-500 dark:text-slate-400 flex items-center">
            <FileText size={12} className="mr-1"/>
              {resource.file_count} {resource.file_count !== 1 ? t.resources.files : t.resources.file}
          </span>
                    <span className={`font-medium ${getTextColor(resource.usage_percent)}`}>
            {resource.usage_percent.toFixed(1)}% {t.resources.used}
          </span>
                </div>
            </div>
        </div>
    );
};

interface ResourceActionsProps {
    resource: FileStorageResource;
    onDelete: (id: string, name: string) => void;
    onViewFiles: (resource: FileStorageResource) => void;
}

const ResourceActions: React.FC<ResourceActionsProps> = (
    {
        resource,
        onDelete,
        onViewFiles,
    }) => {
    const t = useTranslation();
    return (
    <div className="pt-3 border-t border-slate-100 dark:border-slate-800 space-y-2">
        <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
      <span className="flex items-center">
        <Calendar size={12} className="mr-1"/>
        {t.resources.created} {formatShortDate(resource.created_at)}
      </span>
        </div>
        <div className="flex items-center gap-2">
            <Button
                onClick={() => onViewFiles(resource)}
                variant="outline"
                size="sm"
                icon={<Eye size={14}/>}
                className="flex-1 text-blue-600 hover:text-blue-700 hover:bg-blue-50 dark:hover:bg-blue-900/20"
            >
                {t.resources.viewFiles}
            </Button>
            <Button
                onClick={() => onDelete(resource.id, resource.name)}
                variant="ghost"
                size="sm"
                icon={<Trash2 size={14}/>}
                className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
            />
        </div>
    </div>
    );
};
