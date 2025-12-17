import React from 'react';
import {configStyles} from '../../../styles/configStyles';

interface NumberInputProps {
  value: number | undefined;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

/**
 * NumberInput - Number input with min/max/step support
 *
 * Usage:
 * ```tsx
 * <NumberInput
 *   value={temperature}
 *   onChange={setTemperature}
 *   min={0}
 *   max={2}
 *   step={0.1}
 *   placeholder="0.7"
 * />
 * ```
 */
export const NumberInput: React.FC<NumberInputProps> = ({
  value,
  onChange,
  min,
  max,
  step,
  placeholder,
  className,
  disabled = false,
}) => {
  const inputClassName = className || configStyles.input;

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    if (newValue === '') {
      onChange(min ?? 0);
      return;
    }
    const parsed = parseFloat(newValue);
    if (!isNaN(parsed)) {
      onChange(parsed);
    }
  };

  return (
    <input
      type="number"
      value={value ?? ''}
      onChange={handleChange}
      min={min}
      max={max}
      step={step}
      placeholder={placeholder}
      disabled={disabled}
      className={inputClassName}
    />
  );
};
