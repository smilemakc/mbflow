import React from 'react';
import {configStyles} from '../../../styles/configStyles';
import {VariableAutocomplete} from '../../builder/VariableAutocomplete';

interface TextareaProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rows?: number;
  className?: string;
  disabled?: boolean;
  monospace?: boolean;
  enableVariables?: boolean;
}

/**
 * Textarea - Multi-line text input with optional variable autocomplete and monospace font
 *
 * Usage:
 * ```tsx
 * // Regular textarea
 * <Textarea value={value} onChange={setValue} rows={5} />
 *
 * // With variable autocomplete
 * <Textarea value={value} onChange={setValue} enableVariables rows={5} />
 *
 * // Monospace font (for code/JSON)
 * <Textarea value={value} onChange={setValue} monospace rows={8} />
 * ```
 */
export const Textarea: React.FC<TextareaProps> = ({
  value,
  onChange,
  placeholder,
  rows = 4,
  className,
  disabled = false,
  monospace = false,
  enableVariables = false,
}) => {
  const textareaClassName =
    className || (monospace ? configStyles.textareaMonospace : configStyles.textarea);

  if (enableVariables) {
    return (
      <VariableAutocomplete
        type="textarea"
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        rows={rows}
        className={textareaClassName}
      />
    );
  }

  return (
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      rows={rows}
      disabled={disabled}
      className={textareaClassName}
    />
  );
};
