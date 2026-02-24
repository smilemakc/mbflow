import React from 'react';
import { Key, User, Shield, FileText, Settings, Lock } from 'lucide-react';
import { useTranslation } from '@/store/translations';
import { CredentialType, getCredentialTypeLabel } from '@/services/credentialsService';

const CredentialTypeIcon: React.FC<{
  type: CredentialType;
  size?: number;
  className?: string;
}> = ({ type, size = 24, className = '' }) => {
  const iconProps = { size, className };
  switch (type) {
    case 'api_key':
      return <Key {...iconProps} />;
    case 'basic_auth':
      return <User {...iconProps} />;
    case 'oauth2':
      return <Shield {...iconProps} />;
    case 'service_account':
      return <FileText {...iconProps} />;
    case 'custom':
      return <Settings {...iconProps} />;
    default:
      return <Lock {...iconProps} />;
  }
};

interface TypeSelectorProps {
  onSelect: (type: CredentialType) => void;
}

export const TypeSelector: React.FC<TypeSelectorProps> = ({ onSelect }) => {
  const t = useTranslation();

  const types: { type: CredentialType; description: string }[] = [
    { type: 'api_key', description: t.credentials.apiKeyDescription },
    { type: 'basic_auth', description: t.credentials.basicAuthDescription },
    { type: 'oauth2', description: t.credentials.oauth2Description },
    { type: 'service_account', description: t.credentials.serviceAccountDescription },
    { type: 'custom', description: t.credentials.customDescription },
  ];

  return (
    <div className="grid grid-cols-1 gap-3">
      {types.map(({ type, description }) => (
        <button
          key={type}
          onClick={() => onSelect(type)}
          className="flex items-center gap-4 p-4 text-left bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg hover:border-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
        >
          <div className="p-2 bg-slate-200 dark:bg-slate-700 rounded-lg">
            <CredentialTypeIcon type={type} className="text-slate-600 dark:text-slate-300" />
          </div>
          <div>
            <div className="font-medium text-slate-900 dark:text-white">
              {getCredentialTypeLabel(type)}
            </div>
            <div className="text-sm text-slate-500 dark:text-slate-400">{description}</div>
          </div>
        </button>
      ))}
    </div>
  );
};
