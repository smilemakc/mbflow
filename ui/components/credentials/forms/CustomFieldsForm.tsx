import React from 'react';
import { Button } from '@/components/ui';
import { useTranslation } from '@/store/translations';

interface CustomField {
  key: string;
  value: string;
}

interface CustomFieldsFormProps {
  fields: CustomField[];
  onFieldsChange: (fields: CustomField[]) => void;
}

export const CustomFieldsForm: React.FC<CustomFieldsFormProps> = ({ fields, onFieldsChange }) => {
  const t = useTranslation();

  const addField = () => {
    onFieldsChange([...fields, { key: '', value: '' }]);
  };

  const removeField = (index: number) => {
    onFieldsChange(fields.filter((_, i) => i !== index));
  };

  const updateField = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...fields];
    updated[index][field] = value;
    onFieldsChange(updated);
  };

  return (
    <div className="space-y-3">
      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
        {t.credentials.customFields} <span className="text-red-500">*</span>
      </label>
      {fields.map((field, index) => (
        <div key={index} className="flex gap-2">
          <input
            type="text"
            value={field.key}
            onChange={(e) => updateField(index, 'key', e.target.value)}
            placeholder="Key"
            className="flex-1 px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
          />
          <input
            type="password"
            value={field.value}
            onChange={(e) => updateField(index, 'value', e.target.value)}
            placeholder="Value (secret)"
            className="flex-1 px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
          />
          {fields.length > 1 && (
            <Button
              onClick={() => removeField(index)}
              variant="ghost"
              size="sm"
              className="text-red-500"
            >
              ×
            </Button>
          )}
        </div>
      ))}
      <Button onClick={addField} variant="ghost" size="sm">
        + {t.credentials.addField}
      </Button>
    </div>
  );
};
