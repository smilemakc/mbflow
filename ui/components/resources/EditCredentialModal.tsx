/**
 * EditCredentialModal component
 * Single Responsibility: Modal for editing credential name and description
 */

import React, { useState, useEffect } from 'react';
import { Key, User, Shield, FileText, Settings, Lock } from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import { useTranslation } from '@/store/translations';
import { Credential, CredentialType, getCredentialTypeLabel } from '@/services/credentialsService';

const CredentialTypeIcon: React.FC<{ type: CredentialType; className?: string }> = ({ type, className = '' }) => {
  const iconProps = { size: 20, className };
  switch (type) {
    case 'api_key': return <Key {...iconProps} />;
    case 'basic_auth': return <User {...iconProps} />;
    case 'oauth2': return <Shield {...iconProps} />;
    case 'service_account': return <FileText {...iconProps} />;
    case 'custom': return <Settings {...iconProps} />;
    default: return <Lock {...iconProps} />;
  }
};

interface EditCredentialModalProps {
  isOpen: boolean;
  credential: Credential;
  onClose: () => void;
  onSubmit: (id: string, name: string, description?: string) => Promise<boolean>;
}

export const EditCredentialModal: React.FC<EditCredentialModalProps> = ({
  isOpen,
  credential,
  onClose,
  onSubmit,
}) => {
  const t = useTranslation();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen && credential) {
      setName(credential.name);
      setDescription(credential.description || '');
    }
  }, [isOpen, credential]);

  const handleClose = () => {
    if (!loading) {
      onClose();
    }
  };

  const handleSubmit = async () => {
    if (!name.trim()) return;

    setLoading(true);
    const success = await onSubmit(credential.id, name.trim(), description.trim() || undefined);
    setLoading(false);

    if (success) {
      onClose();
    }
  };

  const isFormValid = name.trim().length > 0;

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={t.credentials.editTitle}
      size="md"
      footer={
        <div className="flex justify-end gap-3">
          <Button onClick={handleClose} variant="secondary" disabled={loading}>
            {t.common.cancel}
          </Button>
          <Button
            onClick={handleSubmit}
            variant="primary"
            loading={loading}
            disabled={!isFormValid}
          >
            {t.common.save}
          </Button>
        </div>
      }
    >
      {/* Credential info */}
      <div className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg mb-4">
        <div className="p-2 bg-slate-200 dark:bg-slate-700 rounded-lg">
          <CredentialTypeIcon type={credential.credential_type} className="text-slate-600 dark:text-slate-300" />
        </div>
        <div>
          <div className="text-sm font-medium text-slate-900 dark:text-white">
            {getCredentialTypeLabel(credential.credential_type)}
          </div>
          {credential.provider && (
            <div className="text-xs text-slate-500 dark:text-slate-400">
              {credential.provider}
            </div>
          )}
        </div>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            {t.credentials.name} <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t.credentials.namePlaceholder}
            className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            {t.credentials.description}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t.credentials.descriptionPlaceholder}
            rows={3}
            className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
          />
        </div>
      </div>
    </Modal>
  );
};
