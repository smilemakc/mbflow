/**
 * ResourcesPage - Main page for managing resources and billing
 *
 * SOLID Principles Applied:
 * - Single Responsibility: Each component has one job
 * - Open/Closed: Easy to extend with new resource types
 * - Liskov Substitution: Components are interchangeable via props
 * - Interface Segregation: Props interfaces are minimal
 * - Dependency Inversion: Components depend on abstractions (hooks)
 */

import React, {useState} from 'react';
import {Loader2} from 'lucide-react';

// Hooks
import {useResources} from '@/hooks/resources/useResources.ts';
import {useFileStorage} from '@/hooks/resources/useFileStorage.ts';

// Components
import {
  AccountBalance,
  CreateStorageModal,
  DepositModal,
  FilesModal,
  ResourceList,
  TransactionHistory,
} from '@/components/resources';
import {ConfirmModal} from '@/components/ui';

export const ResourcesPage: React.FC = () => {
    // Data hooks
    const {
        resources,
        account,
        transactions,
        transactionsTotal,
        loading,
        loadData,
        createStorage,
        deleteResource,
        deposit,
    } = useResources();

    const fileStorage = useFileStorage(loadData);

    // Modal states
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showDepositModal, setShowDepositModal] = useState(false);
    const [showFilesModal, setShowFilesModal] = useState(false);
    const [resourceToDelete, setResourceToDelete] = useState<{id: string, name: string} | null>(null);

    // Handlers
    const handleDeleteResource = async () => {
        if (!resourceToDelete) return;
        await deleteResource(resourceToDelete.id);
        setResourceToDelete(null);
    };

    const handleViewFiles = async (resource: any) => {
        await fileStorage.loadFiles(resource);
        setShowFilesModal(true);
    };

    const handleCloseFilesModal = () => {
        setShowFilesModal(false);
        fileStorage.clearSelection();
    };

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-7xl mx-auto space-y-8">
                {/* Header */}
                <PageHeader account={account}/>

                {/* Loading State */}
                {loading && <LoadingSpinner/>}

                {/* Content */}
                {!loading && (
                    <>
                        <ResourceList
                            resources={resources}
                            onCreateClick={() => setShowCreateModal(true)}
                            onDeleteResource={(id, name) => setResourceToDelete({id, name})}
                            onViewFiles={handleViewFiles}
                        />

                        <TransactionHistory
                            transactions={transactions}
                            total={transactionsTotal}
                            onDepositClick={() => setShowDepositModal(true)}
                        />
                    </>
                )}
            </div>

            {/* Modals */}
            <CreateStorageModal
                isOpen={showCreateModal}
                onClose={() => setShowCreateModal(false)}
                onSubmit={createStorage}
            />

            <DepositModal
                isOpen={showDepositModal}
                onClose={() => setShowDepositModal(false)}
                onSubmit={deposit}
            />

            <FilesModal
                isOpen={showFilesModal}
                onClose={handleCloseFilesModal}
                resource={fileStorage.selectedResource}
                files={fileStorage.files}
                filesTotal={fileStorage.filesTotal}
                loading={fileStorage.filesLoading}
                onUpload={fileStorage.uploadFile}
                onDelete={fileStorage.deleteFile}
                onDownload={fileStorage.downloadFile}
            />

            {/* Delete Resource Confirmation Modal */}
            <ConfirmModal
                isOpen={!!resourceToDelete}
                onClose={() => setResourceToDelete(null)}
                onConfirm={handleDeleteResource}
                title="Delete Resource"
                message={`Are you sure you want to delete "${resourceToDelete?.name}"? This action cannot be undone.`}
                confirmText="Delete"
                variant="danger"
            />
        </div>
    );
};

// Sub-components for clarity

interface PageHeaderProps {
    account: any | null;
}

const PageHeader: React.FC<PageHeaderProps> = ({account}) => (
    <div className="flex justify-between items-start">
        <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                Resources & Billing
            </h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">
                Manage your file storage resources and account balance.
            </p>
        </div>
        {account && <AccountBalance account={account}/>}
    </div>
);

const LoadingSpinner: React.FC = () => (
    <div className="flex items-center justify-center py-20">
        <Loader2 size={32} className="animate-spin text-blue-600"/>
    </div>
);

export default ResourcesPage;
