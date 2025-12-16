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
import {useCredentials} from '@/hooks/resources/useCredentials.ts';

// Components
import {
  AccountBalance,
  CreateStorageModal,
  DepositModal,
  FilesModal,
  ResourceList,
  TransactionHistory,
  CredentialList,
  CreateCredentialModal,
  ViewSecretsModal,
} from '@/components/resources';
import {ConfirmModal} from '@/components/ui';
import { useTranslation } from '@/store/translations';
import { Credential, CredentialType } from '@/services/credentialsService';

export const ResourcesPage: React.FC = () => {
    const t = useTranslation();
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

    // Credentials hook
    const {
        credentials,
        loading: credentialsLoading,
        createCredential,
        deleteCredential,
    } = useCredentials();

    // File Storage Modal states
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showDepositModal, setShowDepositModal] = useState(false);
    const [showFilesModal, setShowFilesModal] = useState(false);
    const [resourceToDelete, setResourceToDelete] = useState<{id: string, name: string} | null>(null);

    // Credentials Modal states
    const [showCreateCredentialModal, setShowCreateCredentialModal] = useState(false);
    const [viewingCredential, setViewingCredential] = useState<Credential | null>(null);
    const [credentialToDelete, setCredentialToDelete] = useState<Credential | null>(null);

    // File Storage Handlers
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

    // Credentials Handlers
    const handleCreateCredential = async (type: CredentialType, data: any): Promise<boolean> => {
        const success = await createCredential(type, data);
        if (success) {
            setShowCreateCredentialModal(false);
        }
        return success;
    };

    const handleDeleteCredential = async () => {
        if (!credentialToDelete) return;
        await deleteCredential(credentialToDelete.id);
        setCredentialToDelete(null);
    };

    const isLoading = loading || credentialsLoading;

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-7xl mx-auto space-y-8">
                {/* Header */}
                <PageHeader account={account}/>

                {/* Loading State */}
                {isLoading && <LoadingSpinner/>}

                {/* Content */}
                {!isLoading && (
                    <>
                        {/* File Storage Section */}
                        <ResourceList
                            resources={resources}
                            onCreateClick={() => setShowCreateModal(true)}
                            onDeleteResource={(id, name) => setResourceToDelete({id, name})}
                            onViewFiles={handleViewFiles}
                        />

                        {/* Credentials Section */}
                        <CredentialsSection
                            credentials={credentials}
                            onCreateClick={() => setShowCreateCredentialModal(true)}
                            onViewSecrets={setViewingCredential}
                            onDelete={setCredentialToDelete}
                        />

                        {/* Transactions Section */}
                        <TransactionHistory
                            transactions={transactions}
                            total={transactionsTotal}
                            onDepositClick={() => setShowDepositModal(true)}
                        />
                    </>
                )}
            </div>

            {/* File Storage Modals */}
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
                title={t.resources.deleteResourceTitle}
                message={t.resources.deleteResourceMessage}
                confirmText={t.common.delete}
                variant="danger"
            />

            {/* Credentials Modals */}
            <CreateCredentialModal
                isOpen={showCreateCredentialModal}
                onClose={() => setShowCreateCredentialModal(false)}
                onSubmit={handleCreateCredential}
            />

            {viewingCredential && (
                <ViewSecretsModal
                    isOpen={true}
                    credential={viewingCredential}
                    onClose={() => setViewingCredential(null)}
                />
            )}

            {/* Delete Credential Confirmation Modal */}
            <ConfirmModal
                isOpen={!!credentialToDelete}
                onClose={() => setCredentialToDelete(null)}
                onConfirm={handleDeleteCredential}
                title={t.credentials.deleteTitle}
                message={t.credentials.deleteConfirmation.replace('{name}', credentialToDelete?.name || '')}
                confirmText={t.common.delete}
                variant="danger"
            />
        </div>
    );
};

// Sub-components for clarity

interface PageHeaderProps {
    account: any | null;
}

const PageHeader: React.FC<PageHeaderProps> = ({account}) => {
    const t = useTranslation();
    return (
        <div className="flex justify-between items-start">
            <div>
                <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                    {t.resources.pageTitle}
                </h1>
                <p className="text-slate-500 dark:text-slate-400 mt-1">
                    {t.resources.pageSubtitle}
                </p>
            </div>
            {account && <AccountBalance account={account}/>}
        </div>
    );
};

const LoadingSpinner: React.FC = () => (
    <div className="flex items-center justify-center py-20">
        <Loader2 size={32} className="animate-spin text-blue-600"/>
    </div>
);

interface CredentialsSectionProps {
    credentials: Credential[];
    onCreateClick: () => void;
    onViewSecrets: (credential: Credential) => void;
    onDelete: (credential: Credential) => void;
}

const CredentialsSection: React.FC<CredentialsSectionProps> = ({
    credentials,
    onCreateClick,
    onViewSecrets,
    onDelete,
}) => {
    const t = useTranslation();
    return (
        <CredentialList
            credentials={credentials}
            onCreateClick={onCreateClick}
            onViewSecrets={onViewSecrets}
            onDelete={onDelete}
        />
    );
};

export default ResourcesPage;
