import React from 'react';
import { FormField } from '@/components/ui/form/FormField';
import { TextInput } from '@/components/ui/form/TextInput';
import { useTranslation } from '@/store/translations';

interface BasicAuthFormProps {
  username: string;
  password: string;
  onUsernameChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
}

export const BasicAuthForm: React.FC<BasicAuthFormProps> = ({
  username,
  password,
  onUsernameChange,
  onPasswordChange,
}) => {
  const t = useTranslation();

  return (
    <>
      <FormField label={t.credentials.username} required>
        <TextInput value={username} onChange={onUsernameChange} />
      </FormField>
      <FormField label={t.credentials.password} required>
        <TextInput value={password} onChange={onPasswordChange} type="password" />
      </FormField>
    </>
  );
};
