import React from 'react';
import {configStyles} from '../../../styles/configStyles';

interface SelectOption {
  value: string;
  label: string;
}

interface SelectProps {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[] | string[];
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

/**
 * Select - Dropdown select component
 *
 * Usage:
 * ```tsx
 * // With option objects
 * <Select
 *   value={value}
 *   onChange={setValue}
 *   options={[
 *     { value: 'opt1', label: 'Option 1' },
 *     { value: 'opt2', label: 'Option 2' }
 *   ]}
 * />
 *
 * // With simple string array
 * <Select
 *   value={value}
 *   onChange={setValue}
 *   options={['option1', 'option2', 'option3']}
 * />
 * ```
 */
export const Select: React.FC<SelectProps> = ({
  value,
  onChange,
  options,
  placeholder,
  className,
  disabled = false,
}) => {
  const selectClassName = className || configStyles.select;

  const normalizedOptions: SelectOption[] = options.map((opt) =>
    typeof opt === 'string' ? {value: opt, label: opt} : opt
  );

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className={selectClassName}
    >
      {placeholder && (
        <option value="" disabled>
          {placeholder}
        </option>
      )}
      {normalizedOptions.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
};
