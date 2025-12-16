/**
 * ViewSecretsModal component
 * Single Responsibility: Display decrypted credential secrets
 */

import React, { useState, useEffect } from 'react';
import { Key, User, Shield, FileText, Settings, Lock, Eye, EyeOff, Copy, Check, AlertTriangle } from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import { useTranslation } from '@/store/translations';
import {
  Credential,
  CredentialWithSecrets,
  CredentialType,
  credentialsApi,
  getCredentialTypeLabel,
} from '@/services/credentialsService';

const CredentialTypeIcon: React.FC<{ type: CredentialType; size?: number; className?: string }> = ({ type, size = 20, className = '' }) => {
  const iconProps = { size, className };
  switch (type) {
    case 'api_key': return <Key {...iconProps} />;
    case 'basic_auth': return <User {...iconProps} />;
    case 'oauth2': return <Shield {...iconProps} />;
    case 'service_account': return <FileText {...iconProps} />;
    case 'custom': return <Settings {...iconProps} />;
    default: return <Lock {...iconProps} />;
  }
};

interface ViewSecretsModalProps {
  isOpen: boolean;
  credential: Credential;
  onClose: () => void;
}

export const ViewSecretsModal: React.FC<ViewSecretsModalProps> = ({
  isOpen,
  credential,
  onClose,
}) => {
  const t = useTranslation();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [secrets, setSecrets] = useState<Record<string, string> | null>(null);
  const [visibleFields, setVisibleFields] = useState<Set<string>>(new Set());
  const [copiedField, setCopiedField] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      loadSecrets();
    }
    return () => {
      // Clear secrets when modal closes
      setSecrets(null);
      setVisibleFields(new Set());
    };
  }, [isOpen, credential.id]);

  const loadSecrets = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await credentialsApi.getCredentialSecrets(credential.id);
      setSecrets(response.data.data);
    } catch (err: any) {
      if (err.response?.status === 410) {
        setError(t.credentials.expiredError);
      } else {
        setError(err.response?.data?.error || 'Failed to load secrets');
      }
    } finally {
      setLoading(false);
    }
  };

  const toggleVisibility = (field: string) => {
    const newVisible = new Set(visibleFields);
    if (newVisible.has(field)) {
      newVisible.delete(field);
    } else {
      newVisible.add(field);
    }
    setVisibleFields(newVisible);
  };

  const copyToClipboard = async (field: string, value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const maskValue = (value: string) => {
    if (value.length <= 8) {
      return '••••••••';
    }
    return value.substring(0, 4) + '••••••••' + value.substring(value.length - 4);
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={
        <div className="flex items-center gap-2">
          <CredentialTypeIcon type={credential.credential_type} className="text-slate-600 dark:text-slate-300" />
          <span>{credential.name}</span>
        </div>
      }
      size="lg"
      footer={
        <div className="flex justify-end">
          <Button onClick={onClose} variant="secondary">
            {t.common.close}
          </Button>
        </div>
      }
    >
      {/* Warning banner */}
      <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3 mb-4">
        <div className="flex items-start gap-2">
          <AlertTriangle size={18} className="text-yellow-500 mt-0.5 shrink-0" />
          <div className="text-sm text-yellow-700 dark:text-yellow-300">
            <p className="font-medium">{t.credentials.secretsWarningTitle}</p>
            <p>{t.credentials.secretsWarningText}</p>
          </div>
        </div>
      </div>

      {/* Credential info */}
      <div className="mb-4 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div>
            <span className="text-slate-500 dark:text-slate-400">{t.credentials.type}:</span>{' '}
            <span className="text-slate-900 dark:text-white">
              {getCredentialTypeLabel(credential.credential_type)}
            </span>
          </div>
          {credential.provider && (
            <div>
              <span className="text-slate-500 dark:text-slate-400">{t.credentials.provider}:</span>{' '}
              <span className="text-slate-900 dark:text-white">{credential.provider}</span>
            </div>
          )}
          <div>
            <span className="text-slate-500 dark:text-slate-400">{t.credentials.usageCount}:</span>{' '}
            <span className="text-slate-900 dark:text-white">{credential.usage_count}</span>
          </div>
          {credential.last_used_at && (
            <div>
              <span className="text-slate-500 dark:text-slate-400">{t.credentials.lastUsed}:</span>{' '}
              <span className="text-slate-900 dark:text-white">
                {new Date(credential.last_used_at).toLocaleString()}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Loading state */}
      {loading && (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-700 dark:text-red-300">{error}</p>
          <Button onClick={loadSecrets} variant="ghost" size="sm" className="mt-2">
            {t.common.retry}
          </Button>
        </div>
      )}

      {/* Secrets list */}
      {!loading && !error && secrets && (
        <div className="space-y-3">
          {Object.entries(secrets).map(([field, value]) => (
            <div
              key={field}
              className="border border-slate-200 dark:border-slate-700 rounded-lg p-3"
            >
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
                  {field}
                </label>
                <div className="flex gap-1">
                  <button
                    onClick={() => toggleVisibility(field)}
                    className="p-1 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                    title={visibleFields.has(field) ? 'Hide' : 'Show'}
                  >
                    {visibleFields.has(field) ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                  <button
                    onClick={() => copyToClipboard(field, value)}
                    className="p-1 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                    title="Copy"
                  >
                    {copiedField === field ? <Check size={16} className="text-green-500" /> : <Copy size={16} />}
                  </button>
                </div>
              </div>
              <div className="font-mono text-sm bg-slate-100 dark:bg-slate-800 rounded px-3 py-2 break-all">
                {visibleFields.has(field) ? value : maskValue(value)}
              </div>
            </div>
          ))}
        </div>
      )}
    </Modal>
  );
};
