import React from 'react';
import { Info, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';

export type InfoBoxVariant = 'info' | 'warning' | 'success' | 'error';

interface InfoBoxProps {
  variant?: InfoBoxVariant;
  title?: string;
  children: React.ReactNode;
  icon?: React.ElementType;
}

const VARIANT_STYLES: Record<InfoBoxVariant, { bg: string; border: string; iconColor: string; titleColor: string; defaultIcon: React.ElementType }> = {
  info: {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    border: 'border-blue-200 dark:border-blue-800',
    iconColor: 'text-blue-600 dark:text-blue-400',
    titleColor: 'text-blue-900 dark:text-blue-100',
    defaultIcon: Info,
  },
  warning: {
    bg: 'bg-amber-50 dark:bg-amber-950/20',
    border: 'border-amber-200 dark:border-amber-800',
    iconColor: 'text-amber-600 dark:text-amber-400',
    titleColor: 'text-amber-900 dark:text-amber-100',
    defaultIcon: AlertTriangle,
  },
  success: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    border: 'border-green-200 dark:border-green-800',
    iconColor: 'text-green-600 dark:text-green-400',
    titleColor: 'text-green-900 dark:text-green-100',
    defaultIcon: CheckCircle,
  },
  error: {
    bg: 'bg-red-50 dark:bg-red-900/20',
    border: 'border-red-200 dark:border-red-900/30',
    iconColor: 'text-red-600 dark:text-red-400',
    titleColor: 'text-red-900 dark:text-red-300',
    defaultIcon: XCircle,
  },
};

export const InfoBox: React.FC<InfoBoxProps> = ({
  variant = 'info',
  title,
  children,
  icon,
}) => {
  const styles = VARIANT_STYLES[variant];
  const IconComponent = icon || styles.defaultIcon;

  return (
    <div className={`${styles.bg} border ${styles.border} rounded-lg p-4`}>
      <div className="flex items-start gap-3">
        <IconComponent className={`${styles.iconColor} flex-shrink-0 mt-0.5`} size={16} />
        <div className="flex-1">
          {title && (
            <h4 className={`text-xs font-bold ${styles.titleColor} mb-2`}>{title}</h4>
          )}
          <div className="text-xs text-slate-700 dark:text-slate-300 space-y-1">
            {children}
          </div>
        </div>
      </div>
    </div>
  );
};
