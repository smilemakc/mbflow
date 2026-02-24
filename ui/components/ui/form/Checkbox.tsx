import React from 'react';
import {configStyles} from '../../../styles/configStyles';

interface CheckboxProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
  className?: string;
}

/**
 * Checkbox - Checkbox input with optional label
 *
 * Usage:
 * ```tsx
 * <Checkbox
 *   checked={value}
 *   onChange={setValue}
 *   label="Enable feature"
 * />
 * ```
 */
export const Checkbox: React.FC<CheckboxProps> = ({
  checked,
  onChange,
  label,
  disabled = false,
  className = '',
}) => {
  if (label) {
    return (
      <label className={`flex items-center gap-2 cursor-pointer group ${className}`}>
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          disabled={disabled}
          className={configStyles.checkbox}
        />
        <span
          className={`${configStyles.checkboxLabel} group-hover:text-slate-900 dark:group-hover:text-slate-100 transition-colors`}
        >
          {label}
        </span>
      </label>
    );
  }

  return (
    <input
      type="checkbox"
      checked={checked}
      onChange={(e) => onChange(e.target.checked)}
      disabled={disabled}
      className={`${configStyles.checkbox} ${className}`}
    />
  );
};
