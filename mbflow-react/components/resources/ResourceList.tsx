/**
 * ResourceList component
 * Single Responsibility: Displays list of file storage resources or empty state
 */

import React from 'react';
import {HardDrive, Plus} from 'lucide-react';
import {Button} from '@/components/ui';
import {FileStorageResource} from '@/services/resources.ts';
import {ResourceCard} from './ResourceCard.tsx';
import {useTranslation} from '@/store/translations';

interface ResourceListProps {
    resources: FileStorageResource[];
    onCreateClick: () => void;
    onDeleteResource: (id: string, name: string) => void;
    onViewFiles: (resource: FileStorageResource) => void;
}

export const ResourceList: React.FC<ResourceListProps> = (
    {
        resources,
        onCreateClick,
        onDeleteResource,
        onViewFiles,
    }) => {
    return (
        <section>
            <SectionHeader onCreateClick={onCreateClick}/>
            {resources.length === 0 ? (
                <EmptyState onCreateClick={onCreateClick}/>
            ) : (
                <ResourceGrid
                    resources={resources}
                    onDeleteResource={onDeleteResource}
                    onViewFiles={onViewFiles}
                />
            )}
        </section>
    );
};

interface SectionHeaderProps {
    onCreateClick: () => void;
}

const SectionHeader: React.FC<SectionHeaderProps> = ({onCreateClick}) => {
    const t = useTranslation();
    return (
    <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            {t.resources.fileStorageResources}
        </h2>
        <Button
            onClick={onCreateClick}
            variant="primary"
            size="sm"
            icon={<Plus size={16}/>}
        >
            {t.resources.createStorage}
        </Button>
    </div>
    );
};

interface EmptyStateProps {
    onCreateClick: () => void;
}

const EmptyState: React.FC<EmptyStateProps> = ({onCreateClick}) => {
    const t = useTranslation();
    return (
    <div
        className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-12 text-center">
        <HardDrive size={48} className="mx-auto mb-4 text-slate-300 dark:text-slate-700"/>
        <h3 className="text-lg font-bold text-slate-900 dark:text-white mb-2">
            {t.resources.noStorageYet}
        </h3>
        <p className="text-slate-500 dark:text-slate-400 mb-6">
            {t.resources.noStorageDescription}
        </p>
        <Button
            onClick={onCreateClick}
            variant="primary"
            size="sm"
            icon={<Plus size={16}/>}
        >
            {t.resources.createStorage}
        </Button>
    </div>
    );
};

interface ResourceGridProps {
    resources: FileStorageResource[];
    onDeleteResource: (id: string, name: string) => void;
    onViewFiles: (resource: FileStorageResource) => void;
}

const ResourceGrid: React.FC<ResourceGridProps> = (
    {
        resources,
        onDeleteResource,
        onViewFiles,
    }) => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {resources.map((resource) => (
            <ResourceCard
                key={resource.id}
                resource={resource}
                onDelete={onDeleteResource}
                onViewFiles={onViewFiles}
            />
        ))}
    </div>
);
