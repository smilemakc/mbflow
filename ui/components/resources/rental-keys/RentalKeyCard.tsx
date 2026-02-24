/**
 * RentalKeyCard component
 * Displays a rental key resource with usage statistics and limits.
 * Note: The actual API key value is NEVER shown to users.
 */

import React from 'react';
import {
  AlertCircle,
  Calendar,
  CheckCircle2,
  Clock,
  Eye,
  Key,
  Zap,
} from 'lucide-react';
import { Button } from '@/components/ui';
import {
  RentalKey,
  getProviderLabel,
  formatTokenCount,
  calculateUsagePercent,
  isNearLimit,
} from '@/services/rentalKeyService';
import { formatShortDate } from '@/utils/formatters';
import { useTranslation } from '@/store/translations';

interface RentalKeyCardProps {
  rentalKey: RentalKey;
  onViewUsage: (key: RentalKey) => void;
}

export const RentalKeyCard: React.FC<RentalKeyCardProps> = ({
  rentalKey,
  onViewUsage,
}) => {
  const t = useTranslation();
  const isActive = rentalKey.status === 'active';

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm hover:shadow-md transition-all group">
      <RentalKeyHeader rentalKey={rentalKey} isActive={isActive} />
      <UsageLimits rentalKey={rentalKey} />
      <UsageStats rentalKey={rentalKey} />
      <RentalKeyActions rentalKey={rentalKey} onViewUsage={onViewUsage} />
    </div>
  );
};

interface RentalKeyHeaderProps {
  rentalKey: RentalKey;
  isActive: boolean;
}

const RentalKeyHeader: React.FC<RentalKeyHeaderProps> = ({ rentalKey, isActive }) => {
  const providerColors: Record<string, string> = {
    openai: 'bg-green-50 dark:bg-green-900/20 border-green-100 dark:border-green-900/30 text-green-600 dark:text-green-400',
    anthropic: 'bg-orange-50 dark:bg-orange-900/20 border-orange-100 dark:border-orange-900/30 text-orange-600 dark:text-orange-400',
    google_ai: 'bg-blue-50 dark:bg-blue-900/20 border-blue-100 dark:border-blue-900/30 text-blue-600 dark:text-blue-400',
  };

  const colorClass = providerColors[rentalKey.provider] || 'bg-slate-50 dark:bg-slate-900/20 border-slate-100 dark:border-slate-900/30 text-slate-600 dark:text-slate-400';

  return (
    <div className="flex justify-between items-start mb-4">
      <div className="flex items-start space-x-3 flex-1">
        <div className={`p-2 rounded-lg border ${colorClass}`}>
          <Key size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="font-bold text-slate-900 dark:text-white truncate">
            {rentalKey.name}
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            {getProviderLabel(rentalKey.provider)}
          </p>
          {rentalKey.description && (
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 line-clamp-2">
              {rentalKey.description}
            </p>
          )}
        </div>
      </div>
      <StatusBadge status={rentalKey.status} isActive={isActive} />
    </div>
  );
};

interface StatusBadgeProps {
  status: string;
  isActive: boolean;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({ status, isActive }) => (
  <span
    className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium shrink-0 ml-2 ${
      isActive
        ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400'
        : 'bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-400'
    }`}
  >
    {isActive ? (
      <CheckCircle2 size={10} className="mr-1" />
    ) : (
      <AlertCircle size={10} className="mr-1" />
    )}
    {status}
  </span>
);

interface UsageLimitsProps {
  rentalKey: RentalKey;
}

const UsageLimits: React.FC<UsageLimitsProps> = ({ rentalKey }) => {
  const t = useTranslation();

  const dailyPercent = calculateUsagePercent(rentalKey.requests_today, rentalKey.daily_request_limit);
  const monthlyPercent = calculateUsagePercent(rentalKey.tokens_this_month, rentalKey.monthly_token_limit);

  const dailyNearLimit = isNearLimit(rentalKey.requests_today, rentalKey.daily_request_limit);
  const monthlyNearLimit = isNearLimit(rentalKey.tokens_this_month, rentalKey.monthly_token_limit);

  return (
    <div className="space-y-3 mb-3">
      {/* Daily requests limit */}
      {rentalKey.daily_request_limit && (
        <LimitBar
          label={t.rentalKeys?.dailyRequests || 'Daily Requests'}
          current={rentalKey.requests_today}
          limit={rentalKey.daily_request_limit}
          percent={dailyPercent}
          nearLimit={dailyNearLimit}
          formatter={(v) => v.toString()}
        />
      )}

      {/* Monthly tokens limit */}
      {rentalKey.monthly_token_limit && (
        <LimitBar
          label={t.rentalKeys?.monthlyTokens || 'Monthly Tokens'}
          current={rentalKey.tokens_this_month}
          limit={rentalKey.monthly_token_limit}
          percent={monthlyPercent}
          nearLimit={monthlyNearLimit}
          formatter={formatTokenCount}
        />
      )}
    </div>
  );
};

interface LimitBarProps {
  label: string;
  current: number;
  limit: number;
  percent: number | null;
  nearLimit: boolean;
  formatter: (value: number) => string;
}

const LimitBar: React.FC<LimitBarProps> = ({
  label,
  current,
  limit,
  percent,
  nearLimit,
  formatter,
}) => {
  const getProgressColor = (pct: number | null): string => {
    if (pct === null) return 'bg-blue-600';
    if (pct > 90) return 'bg-red-600';
    if (pct > 75) return 'bg-yellow-500';
    return 'bg-blue-600';
  };

  const getTextColor = (pct: number | null): string => {
    if (pct === null) return 'text-slate-600 dark:text-slate-400';
    if (pct > 90) return 'text-red-600 dark:text-red-400';
    if (pct > 75) return 'text-yellow-600 dark:text-yellow-400';
    return 'text-slate-600 dark:text-slate-400';
  };

  return (
    <div>
      <div className="flex justify-between text-xs mb-1.5">
        <span className="text-slate-600 dark:text-slate-400">{label}</span>
        <span className="font-medium text-slate-900 dark:text-white">
          {formatter(current)} / {formatter(limit)}
        </span>
      </div>
      <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2 overflow-hidden">
        <div
          className={`h-2 rounded-full transition-all ${getProgressColor(percent)}`}
          style={{ width: `${Math.min(percent || 0, 100)}%` }}
        />
      </div>
      {percent !== null && (
        <div className="flex items-center justify-end mt-1 text-xs">
          <span className={`font-medium ${getTextColor(percent)}`}>
            {percent.toFixed(1)}%
          </span>
          {nearLimit && (
            <AlertCircle size={12} className="ml-1 text-yellow-500" />
          )}
        </div>
      )}
    </div>
  );
};

interface UsageStatsProps {
  rentalKey: RentalKey;
}

const UsageStats: React.FC<UsageStatsProps> = ({ rentalKey }) => {
  const t = useTranslation();

  return (
    <div className="grid grid-cols-3 gap-2 py-3 border-t border-b border-slate-100 dark:border-slate-800">
      <div className="text-center">
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          {formatTokenCount(rentalKey.total_requests)}
        </div>
        <div className="text-xs text-slate-500 dark:text-slate-400">
          {t.rentalKeys?.totalRequests || 'Total Requests'}
        </div>
      </div>
      <div className="text-center">
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          {formatTokenCount(rentalKey.total_usage.total)}
        </div>
        <div className="text-xs text-slate-500 dark:text-slate-400">
          {t.rentalKeys?.totalTokens || 'Total Tokens'}
        </div>
      </div>
      <div className="text-center">
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          ${rentalKey.total_cost.toFixed(2)}
        </div>
        <div className="text-xs text-slate-500 dark:text-slate-400">
          {t.rentalKeys?.totalCost || 'Total Cost'}
        </div>
      </div>
    </div>
  );
};

interface RentalKeyActionsProps {
  rentalKey: RentalKey;
  onViewUsage: (key: RentalKey) => void;
}

const RentalKeyActions: React.FC<RentalKeyActionsProps> = ({
  rentalKey,
  onViewUsage,
}) => {
  const t = useTranslation();

  return (
    <div className="pt-3 space-y-2">
      <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
        <span className="flex items-center">
          <Calendar size={12} className="mr-1" />
          {t.resources?.created || 'Created'} {formatShortDate(rentalKey.created_at)}
        </span>
        {rentalKey.last_used_at && (
          <span className="flex items-center">
            <Clock size={12} className="mr-1" />
            {t.rentalKeys?.lastUsed || 'Last used'} {formatShortDate(rentalKey.last_used_at)}
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <Button
          onClick={() => onViewUsage(rentalKey)}
          variant="outline"
          size="sm"
          icon={<Eye size={14} />}
          className="flex-1 text-blue-600 hover:text-blue-700 hover:bg-blue-50 dark:hover:bg-blue-900/20"
        >
          {t.rentalKeys?.viewUsage || 'View Usage'}
        </Button>
      </div>
    </div>
  );
};

export default RentalKeyCard;
