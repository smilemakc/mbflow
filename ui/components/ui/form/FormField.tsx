import React from 'react';
import {configStyles} from '../../../styles/configStyles';

interface FormFieldProps {
  label?: string;
  hint?: string;
  required?: boolean;
  error?: string;
  children: React.ReactNode;
  className?: string;
  htmlFor?: string;
  labelClassName?: string;
}

/**
 * FormField - Wrapper component for form inputs with label, hint, and error support
 *
 * Usage:
 * ```tsx
 * <FormField label="API Key" hint="Enter your API key" required htmlFor="api-key">
 *   <input id="api-key" {...props} />
 * </FormField>
 * ```
 */
export const FormField: React.FC<FormFieldProps> = ({
  label,
  hint,
  required,
  error,
  children,
  className = '',
  htmlFor,
  labelClassName,
}) => (
  <div className={className}>
    {label && (
      <label htmlFor={htmlFor} className={labelClassName || configStyles.label}>
        {label}
        {required && <span className={configStyles.labelRequired}>*</span>}
      </label>
    )}
    {children}
    {hint && !error && <span className={configStyles.hint}>{hint}</span>}
    {error && <span className="text-xs text-red-500 mt-1 block">{error}</span>}
  </div>
);
