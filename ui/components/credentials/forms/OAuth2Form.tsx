import React from 'react';
import { FormField } from '@/components/ui/form/FormField';
import { TextInput } from '@/components/ui/form/TextInput';

interface OAuth2FormProps {
  clientId: string;
  clientSecret: string;
  accessToken: string;
  refreshToken: string;
  tokenUrl: string;
  scopes: string;
  onClientIdChange: (value: string) => void;
  onClientSecretChange: (value: string) => void;
  onAccessTokenChange: (value: string) => void;
  onRefreshTokenChange: (value: string) => void;
  onTokenUrlChange: (value: string) => void;
  onScopesChange: (value: string) => void;
}

export const OAuth2Form: React.FC<OAuth2FormProps> = ({
  clientId,
  clientSecret,
  accessToken,
  refreshToken,
  tokenUrl,
  scopes,
  onClientIdChange,
  onClientSecretChange,
  onAccessTokenChange,
  onRefreshTokenChange,
  onTokenUrlChange,
  onScopesChange,
}) => (
  <>
    <FormField label="Client ID" required>
      <TextInput value={clientId} onChange={onClientIdChange} />
    </FormField>
    <FormField label="Client Secret" required>
      <TextInput value={clientSecret} onChange={onClientSecretChange} type="password" />
    </FormField>
    <FormField label="Access Token">
      <TextInput value={accessToken} onChange={onAccessTokenChange} type="password" />
    </FormField>
    <FormField label="Refresh Token">
      <TextInput value={refreshToken} onChange={onRefreshTokenChange} type="password" />
    </FormField>
    <FormField label="Token URL">
      <TextInput
        value={tokenUrl}
        onChange={onTokenUrlChange}
        placeholder="https://oauth2.example.com/token"
      />
    </FormField>
    <FormField label="Scopes">
      <TextInput value={scopes} onChange={onScopesChange} placeholder="read write" />
    </FormField>
  </>
);
