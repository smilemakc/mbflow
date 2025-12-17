import React from 'react';
import {configStyles} from '../../../styles/configStyles';
import {VariableAutocomplete} from '../../builder/VariableAutocomplete';

interface TextInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  type?: 'text' | 'password' | 'email' | 'url';
  disabled?: boolean;
  enableVariables?: boolean;
  id?: string;
  name?: string;
  autoComplete?: string;
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
}

/**
 * TextInput - Text input component with optional variable autocomplete
 *
 * Usage:
 * ```tsx
 * // Regular input
 * <TextInput value={value} onChange={setValue} placeholder="Enter value" />
 *
 * // With variable autocomplete
 * <TextInput value={value} onChange={setValue} enableVariables placeholder="API key" />
 * ```
 */
export const TextInput: React.FC<TextInputProps> = ({
  value,
  onChange,
  placeholder,
  className,
  type = 'text',
  disabled = false,
  enableVariables = false,
  id,
  name,
  autoComplete,
  required,
  minLength,
  maxLength,
  pattern,
}) => {
  const inputClassName = className || configStyles.input;

  if (enableVariables) {
    return (
      <VariableAutocomplete
        type="input"
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        className={inputClassName}
      />
    );
  }

  return (
    <input
      id={id}
      name={name}
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      autoComplete={autoComplete}
      required={required}
      minLength={minLength}
      maxLength={maxLength}
      pattern={pattern}
      className={inputClassName}
    />
  );
};
