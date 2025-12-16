/**
 * CredentialCard component
 * Single Responsibility: Display a credential resource card
 */

import React from 'react';
import { Key, User, Shield, FileText, Settings, Lock } from 'lucide-react';
import { Button } from '@/components/ui';
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

interface CredentialCardProps {
  credential: Credential;
  onView: () => void;
  onEdit: () => void;
  onDelete: () => void;
}

export const CredentialCard: React.FC<CredentialCardProps> = ({
  credential,
  onView,
  onEdit,
  onDelete,
}) => {
  const t = useTranslation();

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const isExpired = credential.expires_at && new Date(credential.expires_at) < new Date();

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 p-4 hover:shadow-md transition-shadow">
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg">
            <CredentialTypeIcon type={credential.credential_type} className="text-slate-600 dark:text-slate-300" />
          </div>
          <div>
            <h3 className="font-medium text-slate-900 dark:text-white">
              {credential.name}
            </h3>
            <span className="text-xs text-slate-500 dark:text-slate-400">
              {getCredentialTypeLabel(credential.credential_type)}
            </span>
          </div>
        </div>
        <StatusBadge status={credential.status} isExpired={isExpired} />
      </div>

      {/* Description */}
      {credential.description && (
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-3 line-clamp-2">
          {credential.description}
        </p>
      )}

      {/* Provider */}
      {credential.provider && (
        <div className="flex items-center gap-2 mb-3">
          <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
            {t.credentials.provider}:
          </span>
          <span className="text-xs px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded">
            {credential.provider}
          </span>
        </div>
      )}

      {/* Fields */}
      <div className="flex flex-wrap gap-1 mb-3">
        {credential.fields.map((field) => (
          <span
            key={field}
            className="text-xs px-2 py-0.5 bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 rounded"
          >
            {field}
          </span>
        ))}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-2 text-xs text-slate-500 dark:text-slate-400 mb-4">
        <div>
          <span className="font-medium">{t.credentials.usageCount}:</span>{' '}
          {credential.usage_count}
        </div>
        <div>
          <span className="font-medium">{t.credentials.created}:</span>{' '}
          {formatDate(credential.created_at)}
        </div>
        {credential.last_used_at && (
          <div className="col-span-2">
            <span className="font-medium">{t.credentials.lastUsed}:</span>{' '}
            {formatDate(credential.last_used_at)}
          </div>
        )}
        {credential.expires_at && (
          <div className={`col-span-2 ${isExpired ? 'text-red-500' : ''}`}>
            <span className="font-medium">{t.credentials.expires}:</span>{' '}
            {formatDate(credential.expires_at)}
            {isExpired && ` (${t.credentials.expired})`}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <Button onClick={onView} variant="secondary" size="sm" className="flex-1">
          {t.credentials.viewSecrets}
        </Button>
        <Button onClick={onEdit} variant="ghost" size="sm">
          {t.common.edit}
        </Button>
        <Button onClick={onDelete} variant="ghost" size="sm" className="text-red-500 hover:text-red-600">
          {t.common.delete}
        </Button>
      </div>
    </div>
  );
};

interface StatusBadgeProps {
  status: string;
  isExpired?: boolean;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({ status, isExpired }) => {
  const getStatusColor = () => {
    if (isExpired) return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300';
    switch (status) {
      case 'active':
        return 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300';
      case 'suspended':
        return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300';
      case 'deleted':
        return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300';
      default:
        return 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300';
    }
  };

  return (
    <span className={`text-xs px-2 py-1 rounded-full font-medium ${getStatusColor()}`}>
      {isExpired ? 'expired' : status}
    </span>
  );
};
