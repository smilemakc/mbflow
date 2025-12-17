import React from 'react';

export type StatusVariant = 'success' | 'error' | 'warning' | 'info' | 'neutral';
export type StatusSize = 'sm' | 'md' | 'lg';

interface StatusVariantConfig {
  bg: string;
  text: string;
  border: string;
  icon?: React.ReactNode;
}

interface StatusBadgeProps {
  status: string;
  variant: StatusVariant | StatusVariantConfig;
  size?: StatusSize;
  icon?: React.ReactNode;
}

const VARIANT_STYLES: Record<StatusVariant, StatusVariantConfig> = {
  success: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    text: 'text-green-700 dark:text-green-400',
    border: 'border-green-200 dark:border-green-900/30',
  },
  error: {
    bg: 'bg-red-50 dark:bg-red-900/20',
    text: 'text-red-700 dark:text-red-400',
    border: 'border-red-200 dark:border-red-900/30',
  },
  warning: {
    bg: 'bg-yellow-50 dark:bg-yellow-900/20',
    text: 'text-yellow-700 dark:text-yellow-400',
    border: 'border-yellow-200 dark:border-yellow-900/30',
  },
  info: {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    text: 'text-blue-700 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-900/30',
  },
  neutral: {
    bg: 'bg-slate-50 dark:bg-slate-900/20',
    text: 'text-slate-700 dark:text-slate-400',
    border: 'border-slate-200 dark:border-slate-800',
  },
};

const SIZE_STYLES: Record<StatusSize, { padding: string; text: string; iconSize: string; gap: string }> = {
  sm: {
    padding: 'px-2 py-0.5',
    text: 'text-xs',
    iconSize: 'w-3 h-3',
    gap: 'gap-1',
  },
  md: {
    padding: 'px-2.5 py-1',
    text: 'text-xs',
    iconSize: 'w-3.5 h-3.5',
    gap: 'gap-1.5',
  },
  lg: {
    padding: 'px-3 py-1.5',
    text: 'text-sm',
    iconSize: 'w-4 h-4',
    gap: 'gap-2',
  },
};

export const StatusBadge: React.FC<StatusBadgeProps> = ({
  status,
  variant,
  size = 'md',
  icon,
}) => {
  const variantConfig = typeof variant === 'string' ? VARIANT_STYLES[variant] : variant;
  const sizeConfig = SIZE_STYLES[size];

  const displayIcon = icon || variantConfig.icon;

  return (
    <span
      className={`inline-flex items-center ${sizeConfig.gap} ${sizeConfig.padding} rounded-full ${sizeConfig.text} font-medium border ${variantConfig.bg} ${variantConfig.text} ${variantConfig.border}`}
    >
      {displayIcon && <span className={sizeConfig.iconSize}>{displayIcon}</span>}
      {status}
    </span>
  );
};
