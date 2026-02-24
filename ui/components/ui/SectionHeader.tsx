import React from 'react';

export type SectionHeaderVariant = 'blue' | 'green' | 'amber' | 'orange' | 'cyan' | 'purple' | 'slate';

interface SectionHeaderProps {
  icon: React.ElementType;
  title: string;
  description: string;
  variant?: SectionHeaderVariant;
}

const VARIANT_STYLES: Record<SectionHeaderVariant, { gradient: string; iconColor: string; border: string }> = {
  blue: {
    gradient: 'bg-gradient-to-r from-blue-50 to-cyan-50 dark:from-blue-900/10 dark:to-cyan-900/10',
    iconColor: 'text-blue-600 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-800',
  },
  green: {
    gradient: 'bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/10 dark:to-emerald-900/10',
    iconColor: 'text-green-600 dark:text-green-400',
    border: 'border-green-200 dark:border-green-800',
  },
  amber: {
    gradient: 'bg-gradient-to-r from-amber-50 to-yellow-50 dark:from-amber-900/10 dark:to-yellow-900/10',
    iconColor: 'text-amber-600 dark:text-amber-400',
    border: 'border-amber-200 dark:border-amber-800',
  },
  orange: {
    gradient: 'bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-900/10 dark:to-amber-900/10',
    iconColor: 'text-orange-600 dark:text-orange-400',
    border: 'border-orange-200 dark:border-orange-800',
  },
  cyan: {
    gradient: 'bg-gradient-to-r from-cyan-50 to-sky-50 dark:from-cyan-900/10 dark:to-sky-900/10',
    iconColor: 'text-cyan-600 dark:text-cyan-400',
    border: 'border-cyan-200 dark:border-cyan-800',
  },
  purple: {
    gradient: 'bg-gradient-to-r from-purple-50 to-pink-50 dark:from-purple-900/10 dark:to-pink-900/10',
    iconColor: 'text-purple-600 dark:text-purple-400',
    border: 'border-purple-200 dark:border-purple-800',
  },
  slate: {
    gradient: 'bg-gradient-to-r from-slate-50 to-gray-50 dark:from-slate-900/10 dark:to-gray-900/10',
    iconColor: 'text-slate-600 dark:text-slate-400',
    border: 'border-slate-200 dark:border-slate-800',
  },
};

export const SectionHeader: React.FC<SectionHeaderProps> = ({
  icon: Icon,
  title,
  description,
  variant = 'blue',
}) => {
  const styles = VARIANT_STYLES[variant];

  return (
    <div
      className={`${styles.gradient} border ${styles.border} rounded-lg p-4 flex items-start gap-3`}
    >
      <Icon className={`${styles.iconColor} flex-shrink-0 mt-0.5`} size={18} />
      <div>
        <h3 className="font-semibold text-slate-900 dark:text-white text-sm">{title}</h3>
        <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">{description}</p>
      </div>
    </div>
  );
};
