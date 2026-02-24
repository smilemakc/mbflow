import React from 'react';
import { FormField } from '@/components/ui/form/FormField';
import { Textarea } from '@/components/ui/form/Textarea';

interface ServiceAccountFormProps {
  jsonKey: string;
  onJsonKeyChange: (value: string) => void;
}

export const ServiceAccountForm: React.FC<ServiceAccountFormProps> = ({
  jsonKey,
  onJsonKeyChange,
}) => (
  <FormField label="JSON Key" required>
    <Textarea
      value={jsonKey}
      onChange={onJsonKeyChange}
      placeholder='{"type": "service_account", ...}'
      rows={8}
      monospace
    />
  </FormField>
);
