import React from 'react';
import { FormField } from '@/components/ui/form/FormField';
import { TextInput } from '@/components/ui/form/TextInput';

interface ApiKeyFormProps {
  apiKey: string;
  onApiKeyChange: (value: string) => void;
}

export const ApiKeyForm: React.FC<ApiKeyFormProps> = ({ apiKey, onApiKeyChange }) => (
  <FormField label="API Key" required>
    <TextInput
      value={apiKey}
      onChange={onApiKeyChange}
      placeholder="sk-..."
      type="password"
    />
  </FormField>
);
