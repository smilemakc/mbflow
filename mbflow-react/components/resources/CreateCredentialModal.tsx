/**
 * CreateCredentialModal component
 * Single Responsibility: Modal for creating new credential resources
 */

import React, { useState } from 'react';
import { Button, Modal } from '@/components/ui';
import { FormField } from '@/components/ui/form/FormField';
import { TextInput } from '@/components/ui/form/TextInput';
import { Select } from '@/components/ui/form/Select';
import { Textarea } from '@/components/ui/form/Textarea';
import { useTranslation } from '@/store/translations';
import { TypeSelector } from '@/components/credentials/TypeSelector';
import {
  ApiKeyForm,
  BasicAuthForm,
  OAuth2Form,
  ServiceAccountForm,
  CustomFieldsForm,
} from '@/components/credentials/forms';
import { useCredentialForm } from '@/hooks/useCredentialForm';
import { CredentialType, COMMON_PROVIDERS, getCredentialTypeLabel } from '@/services/credentialsService';

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
  const { state, setField, reset, buildRequest, isValid } = useCredentialForm();

  const handleClose = () => {
    if (!loading) {
      reset();
      setStep('select-type');
      setSelectedType(null);
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
    if (!selectedType) return;

    setLoading(true);
    const data = buildRequest(selectedType);
    const success = await onSubmit(selectedType, data);
    setLoading(false);

    if (success) {
      reset();
      setStep('select-type');
      setSelectedType(null);
      onClose();
    }
  };

  const renderTypeSpecificForm = () => {
    switch (selectedType) {
      case 'api_key':
        return (
          <ApiKeyForm
            apiKey={state.apiKey}
            onApiKeyChange={(value) => setField('apiKey', value)}
          />
        );

      case 'basic_auth':
        return (
          <BasicAuthForm
            username={state.username}
            password={state.password}
            onUsernameChange={(value) => setField('username', value)}
            onPasswordChange={(value) => setField('password', value)}
          />
        );

      case 'oauth2':
        return (
          <OAuth2Form
            clientId={state.clientId}
            clientSecret={state.clientSecret}
            accessToken={state.accessToken}
            refreshToken={state.refreshToken}
            tokenUrl={state.tokenUrl}
            scopes={state.scopes}
            onClientIdChange={(value) => setField('clientId', value)}
            onClientSecretChange={(value) => setField('clientSecret', value)}
            onAccessTokenChange={(value) => setField('accessToken', value)}
            onRefreshTokenChange={(value) => setField('refreshToken', value)}
            onTokenUrlChange={(value) => setField('tokenUrl', value)}
            onScopesChange={(value) => setField('scopes', value)}
          />
        );

      case 'service_account':
        return (
          <ServiceAccountForm
            jsonKey={state.jsonKey}
            onJsonKeyChange={(value) => setField('jsonKey', value)}
          />
        );

      case 'custom':
        return (
          <CustomFieldsForm
            fields={state.customFields}
            onFieldsChange={(fields) => setField('customFields', fields)}
          />
        );

      default:
        return null;
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={
        step === 'select-type'
          ? t.credentials.createTitle
          : `${t.credentials.createTitle} - ${selectedType ? getCredentialTypeLabel(selectedType) : ''}`
      }
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
                disabled={!isValid(selectedType)}
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
          <FormField label={t.credentials.name} required>
            <TextInput
              value={state.name}
              onChange={(value) => setField('name', value)}
              placeholder={t.credentials.namePlaceholder}
            />
          </FormField>

          <FormField label={t.credentials.description}>
            <Textarea
              value={state.description}
              onChange={(value) => setField('description', value)}
              placeholder={t.credentials.descriptionPlaceholder}
              rows={3}
            />
          </FormField>

          <FormField label={t.credentials.provider}>
            <Select
              value={state.provider}
              onChange={(value) => setField('provider', value)}
              options={COMMON_PROVIDERS}
              placeholder="Select provider (optional)"
            />
          </FormField>

          {renderTypeSpecificForm()}
        </div>
      )}
    </Modal>
  );
};
