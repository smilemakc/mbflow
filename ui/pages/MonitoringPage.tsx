import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Activity, AlertTriangle, CheckCircle2, Clock, Loader2, PlayCircle, RefreshCw, XCircle } from 'lucide-react';
import { Button } from '@/components/ui';
import { HealthPanel } from '@/components/system/HealthPanel';
import { useSystemOverview } from '@/hooks/useSystemOverview';
import { formatDuration, formatPercent, formatTimeAgo, type ExecutionStatusName } from '@/lib/executionStats';
import { useTranslation } from '@/store/translations';

const REFRESH_MS = 30_000;

const STATUS_COLORS: Record<ExecutionStatusName, string> = {
  completed: 'bg-green-500',
  failed: 'bg-red-500',
  timeout: 'bg-orange-500',
  cancelled: 'bg-slate-400',
  running: 'bg-blue-500',
  pending: 'bg-yellow-400',
};

export const MonitoringPage: React.FC = () => {
  const t = useTranslation();
  const s = t.system;
  const navigate = useNavigate();
  const { health, stats, loading, error, refresh } = useSystemOverview({ sampleSize: 100, refreshMs: REFRESH_MS });

  const timeLabels = { justNow: s.justNow, minutesAgo: s.minutesAgo, hoursAgo: s.hoursAgo, daysAgo: s.daysAgo };
  const statuses = Object.keys(STATUS_COLORS) as ExecutionStatusName[];

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex justify-between items-end gap-4">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{s.title}</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">{s.subtitle}</p>
          </div>
          <div className="flex items-center gap-3">
            {health && (
              <span className="text-xs text-slate-400">
                {s.lastChecked.replace('{time}', health.checkedAt.toLocaleTimeString())}
              </span>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => void refresh()}
              disabled={loading}
              icon={loading ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={16} />}
            >
              {s.refresh}
            </Button>
          </div>
        </div>

        {/* Execution metrics computed from the latest executions */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard title={s.active} value={stats ? String(stats.active) : '-'} icon={<PlayCircle size={20} />} tone="blue" />
          <MetricCard
            title={s.failed}
            value={stats ? String(stats.byStatus.failed + stats.byStatus.timeout) : '-'}
            icon={<XCircle size={20} />}
            tone="red"
          />
          <MetricCard
            title={s.successRate}
            value={stats ? formatPercent(stats.successRate) : '-'}
            icon={<CheckCircle2 size={20} />}
            tone="green"
          />
          <MetricCard
            title={s.avgDuration}
            value={stats ? formatDuration(stats.avgDurationMs) : '-'}
            icon={<Clock size={20} />}
            tone="purple"
          />
        </div>

        {stats && stats.sampleSize > 0 && (
          <p className="text-xs text-slate-400 -mt-2">
            {s.sampleNote.replace('{n}', String(stats.sampleSize)).replace('{total}', String(stats.total))}
          </p>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            {/* Status breakdown */}
            <section className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm p-5">
              <h3 className="font-bold text-slate-800 dark:text-slate-100 mb-4 flex items-center">
                <Activity size={18} className="mr-2 text-slate-400" />
                {s.statusBreakdown}
              </h3>
              {error ? (
                <p role="alert" className="text-sm text-red-600 dark:text-red-400">
                  {s.executionsError}: {error}
                </p>
              ) : !stats ? (
                <p className="text-sm text-slate-400">{s.checking}</p>
              ) : stats.sampleSize === 0 ? (
                <p className="text-sm text-slate-400">{s.noExecutions}</p>
              ) : (
                <>
                  <div className="flex h-3 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                    {statuses.map((status) =>
                      stats.byStatus[status] > 0 ? (
                        <div
                          key={status}
                          className={STATUS_COLORS[status]}
                          style={{ width: `${(stats.byStatus[status] / stats.sampleSize) * 100}%` }}
                          title={`${s.statuses[status]}: ${stats.byStatus[status]}`}
                        />
                      ) : null,
                    )}
                  </div>
                  <ul className="mt-4 grid grid-cols-2 sm:grid-cols-3 gap-2 text-sm">
                    {statuses.map((status) => (
                      <li key={status} className="flex items-center justify-between rounded-lg bg-slate-50 dark:bg-slate-800/50 px-3 py-2">
                        <span className="flex items-center text-slate-600 dark:text-slate-300">
                          <span className={`w-2 h-2 rounded-full mr-2 ${STATUS_COLORS[status]}`} aria-hidden="true" />
                          {s.statuses[status]}
                        </span>
                        <span className="font-semibold text-slate-800 dark:text-slate-100" data-testid={`count-${status}`}>
                          {stats.byStatus[status]}
                        </span>
                      </li>
                    ))}
                  </ul>
                </>
              )}
            </section>

            {/* Recent failures */}
            <section className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm">
              <h3 className="p-4 border-b border-slate-200 dark:border-slate-800 font-bold text-slate-800 dark:text-slate-100 flex items-center">
                <AlertTriangle size={16} className="mr-2 text-slate-500" />
                {s.recentFailures}
              </h3>
              {stats && stats.recentFailures.length > 0 ? (
                <ul className="divide-y divide-slate-100 dark:divide-slate-800">
                  {stats.recentFailures.map((execution) => (
                    <li key={execution.id}>
                      <button
                        type="button"
                        onClick={() => navigate(`/executions/${execution.id}`)}
                        className="w-full text-left px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800/50"
                      >
                        <div className="flex justify-between text-sm">
                          <span className="font-medium text-slate-800 dark:text-slate-200">
                            {execution.workflow_name || execution.workflow_id}
                          </span>
                          <span className="text-xs text-slate-400">{formatTimeAgo(execution.started_at, timeLabels)}</span>
                        </div>
                        {execution.error && (
                          <p className="mt-1 text-xs font-mono text-red-600 dark:text-red-400 break-all line-clamp-2">
                            {execution.error}
                          </p>
                        )}
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="p-4 text-sm text-slate-400">{stats ? s.noFailures : s.checking}</p>
              )}
            </section>
          </div>

          <section className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm p-5 h-fit">
            <h3 className="font-bold text-slate-800 dark:text-slate-100 mb-4 flex items-center">
              <Activity size={18} className="mr-2 text-slate-400" />
              {s.health}
            </h3>
            <HealthPanel health={health} />
          </section>
        </div>
      </div>
    </div>
  );
};

const TONES = {
  blue: 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400',
  red: 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400',
  green: 'bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400',
  purple: 'bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400',
} as const;

const MetricCard: React.FC<{ title: string; value: string; icon: React.ReactNode; tone: keyof typeof TONES }> = ({
  title,
  value,
  icon,
  tone,
}) => (
  <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm">
    <div className="flex justify-between items-start">
      <div>
        <h3 className="text-slate-500 dark:text-slate-400 text-sm font-medium">{title}</h3>
        <div className="text-2xl font-bold text-slate-900 dark:text-white mt-1">{value}</div>
      </div>
      <div className={`p-2 rounded-lg ${TONES[tone]}`}>{icon}</div>
    </div>
  </div>
);
