/**
 * CreateCredentialModal component
 * Single Responsibility: Modal for creating new credential resources
 */

import React, { useState } from 'react';
import { Key, User, Shield, FileText, Settings, Lock } from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import { useTranslation } from '@/store/translations';
import {
  CredentialType,
  CreateAPIKeyRequest,
  CreateBasicAuthRequest,
  CreateOAuth2Request,
  CreateServiceAccountRequest,
  CreateCustomCredentialRequest,
  COMMON_PROVIDERS,
  getCredentialTypeLabel,
} from '@/services/credentialsService';

const CredentialTypeIcon: React.FC<{ type: CredentialType; size?: number; className?: string }> = ({ type, size = 24, className = '' }) => {
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

interface CreateCredentialModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (type: CredentialType, data: any) => Promise<boolean>;
}

type Step = 'select-type' | 'fill-form';

export const CreateCredentialModal: React.FC<CreateCredentialModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
}) => {
  const t = useTranslation();
  const [step, setStep] = useState<Step>('select-type');
  const [selectedType, setSelectedType] = useState<CredentialType | null>(null);
  const [loading, setLoading] = useState(false);

  // Common fields
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [provider, setProvider] = useState('');

  // Type-specific fields
  const [apiKey, setApiKey] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [accessToken, setAccessToken] = useState('');
  const [refreshToken, setRefreshToken] = useState('');
  const [tokenUrl, setTokenUrl] = useState('');
  const [scopes, setScopes] = useState('');
  const [jsonKey, setJsonKey] = useState('');
  const [customFields, setCustomFields] = useState<{ key: string; value: string }[]>([
    { key: '', value: '' },
  ]);

  const resetForm = () => {
    setStep('select-type');
    setSelectedType(null);
    setName('');
    setDescription('');
    setProvider('');
    setApiKey('');
    setUsername('');
    setPassword('');
    setClientId('');
    setClientSecret('');
    setAccessToken('');
    setRefreshToken('');
    setTokenUrl('');
    setScopes('');
    setJsonKey('');
    setCustomFields([{ key: '', value: '' }]);
  };

  const handleClose = () => {
    if (!loading) {
      resetForm();
      onClose();
    }
  };

  const handleSelectType = (type: CredentialType) => {
    setSelectedType(type);
    setStep('fill-form');
  };

  const handleBack = () => {
    setStep('select-type');
  };

  const handleSubmit = async () => {
    if (!selectedType || !name.trim()) return;

    setLoading(true);

    let data: any;
    switch (selectedType) {
      case 'api_key':
        data = {
          name: name.trim(),
          description: description.trim(),
          provider: provider.trim(),
          api_key: apiKey,
        } as CreateAPIKeyRequest;
        break;
      case 'basic_auth':
        data = {
          name: name.trim(),
          description: description.trim(),
          provider: provider.trim(),
          username,
          password,
        } as CreateBasicAuthRequest;
        break;
      case 'oauth2':
        data = {
          name: name.trim(),
          description: description.trim(),
          provider: provider.trim(),
          client_id: clientId,
          client_secret: clientSecret,
          access_token: accessToken || undefined,
          refresh_token: refreshToken || undefined,
          token_url: tokenUrl || undefined,
          scopes: scopes || undefined,
        } as CreateOAuth2Request;
        break;
      case 'service_account':
        data = {
          name: name.trim(),
          description: description.trim(),
          provider: provider.trim(),
          json_key: jsonKey,
        } as CreateServiceAccountRequest;
        break;
      case 'custom':
        const customData: Record<string, string> = {};
        customFields.forEach((f) => {
          if (f.key.trim() && f.value) {
            customData[f.key.trim()] = f.value;
          }
        });
        data = {
          name: name.trim(),
          description: description.trim(),
          provider: provider.trim(),
          data: customData,
        } as CreateCustomCredentialRequest;
        break;
    }

    const success = await onSubmit(selectedType, data);
    setLoading(false);

    if (success) {
      resetForm();
      onClose();
    }
  };

  const isFormValid = (): boolean => {
    if (!name.trim()) return false;

    switch (selectedType) {
      case 'api_key':
        return !!apiKey;
      case 'basic_auth':
        return !!username && !!password;
      case 'oauth2':
        return !!clientId && !!clientSecret;
      case 'service_account':
        return !!jsonKey;
      case 'custom':
        return customFields.some((f) => f.key.trim() && f.value);
      default:
        return false;
    }
  };

  const addCustomField = () => {
    setCustomFields([...customFields, { key: '', value: '' }]);
  };

  const removeCustomField = (index: number) => {
    setCustomFields(customFields.filter((_, i) => i !== index));
  };

  const updateCustomField = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...customFields];
    updated[index][field] = value;
    setCustomFields(updated);
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={step === 'select-type' ? t.credentials.createTitle : t.credentials.createTitle + ` - ${selectedType ? getCredentialTypeLabel(selectedType) : ''}`}
      size="lg"
      footer={
        step === 'fill-form' && (
          <div className="flex justify-between">
            <Button onClick={handleBack} variant="ghost" disabled={loading}>
              {t.common.back}
            </Button>
            <div className="flex gap-3">
              <Button onClick={handleClose} variant="secondary" disabled={loading}>
                {t.common.cancel}
              </Button>
              <Button
                onClick={handleSubmit}
                variant="primary"
                loading={loading}
                disabled={!isFormValid()}
              >
                {t.credentials.create}
              </Button>
            </div>
          </div>
        )
      }
    >
      {step === 'select-type' ? (
        <TypeSelector onSelect={handleSelectType} />
      ) : (
        <div className="space-y-4">
          {/* Common fields */}
          <FormField
            label={t.credentials.name}
            required
            value={name}
            onChange={setName}
            placeholder={t.credentials.namePlaceholder}
          />
          <FormTextArea
            label={t.credentials.description}
            value={description}
            onChange={setDescription}
            placeholder={t.credentials.descriptionPlaceholder}
          />
          <FormSelect
            label={t.credentials.provider}
            value={provider}
            onChange={setProvider}
            options={COMMON_PROVIDERS}
          />

          {/* Type-specific fields */}
          {selectedType === 'api_key' && (
            <FormField
              label="API Key"
              required
              value={apiKey}
              onChange={setApiKey}
              placeholder="sk-..."
              type="password"
            />
          )}

          {selectedType === 'basic_auth' && (
            <>
              <FormField
                label={t.credentials.username}
                required
                value={username}
                onChange={setUsername}
              />
              <FormField
                label={t.credentials.password}
                required
                value={password}
                onChange={setPassword}
                type="password"
              />
            </>
          )}

          {selectedType === 'oauth2' && (
            <>
              <FormField
                label="Client ID"
                required
                value={clientId}
                onChange={setClientId}
              />
              <FormField
                label="Client Secret"
                required
                value={clientSecret}
                onChange={setClientSecret}
                type="password"
              />
              <FormField
                label="Access Token"
                value={accessToken}
                onChange={setAccessToken}
                type="password"
              />
              <FormField
                label="Refresh Token"
                value={refreshToken}
                onChange={setRefreshToken}
                type="password"
              />
              <FormField
                label="Token URL"
                value={tokenUrl}
                onChange={setTokenUrl}
                placeholder="https://oauth2.example.com/token"
              />
              <FormField
                label="Scopes"
                value={scopes}
                onChange={setScopes}
                placeholder="read write"
              />
            </>
          )}

          {selectedType === 'service_account' && (
            <FormTextArea
              label="JSON Key"
              required
              value={jsonKey}
              onChange={setJsonKey}
              placeholder='{"type": "service_account", ...}'
              rows={8}
            />
          )}

          {selectedType === 'custom' && (
            <div className="space-y-3">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
                {t.credentials.customFields} <span className="text-red-500">*</span>
              </label>
              {customFields.map((field, index) => (
                <div key={index} className="flex gap-2">
                  <input
                    type="text"
                    value={field.key}
                    onChange={(e) => updateCustomField(index, 'key', e.target.value)}
                    placeholder="Key"
                    className="flex-1 px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
                  />
                  <input
                    type="password"
                    value={field.value}
                    onChange={(e) => updateCustomField(index, 'value', e.target.value)}
                    placeholder="Value (secret)"
                    className="flex-1 px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
                  />
                  {customFields.length > 1 && (
                    <Button
                      onClick={() => removeCustomField(index)}
                      variant="ghost"
                      size="sm"
                      className="text-red-500"
                    >
                      ×
                    </Button>
                  )}
                </div>
              ))}
              <Button onClick={addCustomField} variant="ghost" size="sm">
                + {t.credentials.addField}
              </Button>
            </div>
          )}
        </div>
      )}
    </Modal>
  );
};

// TypeSelector component
interface TypeSelectorProps {
  onSelect: (type: CredentialType) => void;
}

const TypeSelector: React.FC<TypeSelectorProps> = ({ onSelect }) => {
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

// Form components
interface FormFieldProps {
  label: string;
  required?: boolean;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: string;
}

const FormField: React.FC<FormFieldProps> = ({
  label,
  required,
  value,
  onChange,
  placeholder,
  type = 'text',
}) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
      {label} {required && <span className="text-red-500">*</span>}
    </label>
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
    />
  </div>
);

interface FormTextAreaProps {
  label: string;
  required?: boolean;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rows?: number;
}

const FormTextArea: React.FC<FormTextAreaProps> = ({
  label,
  required,
  value,
  onChange,
  placeholder,
  rows = 3,
}) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
      {label} {required && <span className="text-red-500">*</span>}
    </label>
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      rows={rows}
      className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 font-mono"
    />
  </div>
);

interface FormSelectProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
}

const FormSelect: React.FC<FormSelectProps> = ({ label, value, onChange, options }) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
      {label}
    </label>
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
    >
      <option value="">Select provider (optional)</option>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  </div>
);
