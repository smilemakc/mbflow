import React, { useEffect, useMemo, useState } from 'react';
import { Activity, AlertCircle, CheckCircle2, Clock, Loader2, Server, Workflow, XCircle, Zap } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useUIStore } from '@/store/uiStore';
import { useTranslation } from '@/store/translations';
import { workflowService } from '@/services/workflowService';
import { Button } from '@/components/ui';
import { HealthPanel } from '@/components/system/HealthPanel';
import { useSystemOverview } from '@/hooks/useSystemOverview';
import {
  executionDurationMs,
  formatDuration,
  formatPercent,
  formatTimeAgo,
  type ExecutionLike,
} from '@/lib/executionStats';

const TONES = {
  blue: 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400',
  orange: 'bg-orange-50 dark:bg-orange-900/20 text-orange-600 dark:text-orange-400',
  green: 'bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400',
  purple: 'bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400',
} as const;

const RECENT_COUNT = 5;

export const DashboardPage: React.FC = () => {
  const { setActiveModal } = useUIStore();
  const navigate = useNavigate();
  const t = useTranslation();

  const [workflowNames, setWorkflowNames] = useState<Map<string, string> | null>(null);
  const { health, stats, executions, loading } = useSystemOverview({ sampleSize: 100 });

  useEffect(() => {
    let cancelled = false;
    workflowService
      .getAll()
      .then((list) => {
        if (!cancelled) setWorkflowNames(new Map(list.map((wf) => [wf.id, wf.name])));
      })
      .catch((error) => {
        console.error('Failed to fetch workflows:', error);
        if (!cancelled) setWorkflowNames(new Map());
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const recentRuns = useMemo(() => executions.slice(0, RECENT_COUNT), [executions]);

  const handleCreateNew = () => {
    setActiveModal('templates');
    navigate('/builder');
  };

  const sampleHint = stats ? t.dashboard.lastNExecutions.replace('{n}', String(stats.sampleSize)) : undefined;

  const statCards = [
    {
      label: t.dashboard.totalWorkflows,
      value: workflowNames ? String(workflowNames.size) : '-',
      icon: Workflow,
      tone: 'blue' as const,
    },
    {
      label: t.dashboard.executionsToday,
      value: stats ? `${stats.today}${stats.todayIsLowerBound ? '+' : ''}` : '-',
      icon: Zap,
      tone: 'orange' as const,
    },
    {
      label: t.dashboard.successRate,
      value: stats ? formatPercent(stats.successRate) : '-',
      icon: CheckCircle2,
      tone: 'green' as const,
      hint: sampleHint,
    },
    {
      label: t.dashboard.avgDuration,
      value: stats ? formatDuration(stats.avgDurationMs) : '-',
      icon: Clock,
      tone: 'purple' as const,
      hint: sampleHint,
    },
  ];

  const timeLabels = {
    justNow: t.system.justNow,
    minutesAgo: t.system.minutesAgo,
    hoursAgo: t.system.hoursAgo,
    daysAgo: t.system.daysAgo,
  };

  const workflowLabel = (run: ExecutionLike) =>
    run.workflow_name || workflowNames?.get(run.workflow_id) || `${run.workflow_id.substring(0, 8)}...`;

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t.dashboard.title}</h1>
          <p className="text-slate-500 dark:text-slate-400 mt-1">{t.dashboard.subtitle}</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {statCards.map((stat) => (
            <div
              key={stat.label}
              className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-start">
                <div className={`p-2 rounded-lg ${TONES[stat.tone]}`}>
                  <stat.icon size={20} />
                </div>
                {stat.hint && <span className="text-[10px] text-slate-400">{stat.hint}</span>}
              </div>
              <div className="mt-4">
                <h3 className="text-2xl font-bold text-slate-900 dark:text-white">{stat.value}</h3>
                <p className="text-sm text-slate-500 dark:text-slate-400 font-medium">{stat.label}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden flex flex-col">
            <div className="p-5 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center bg-slate-50/50 dark:bg-slate-900/50">
              <h2 className="font-bold text-slate-800 dark:text-slate-100 flex items-center">
                <Activity size={18} className="mr-2 text-slate-400" />
                {t.dashboard.recentActivity}
              </h2>
              <Button variant="ghost" size="sm" onClick={() => navigate('/executions')}>
                {t.dashboard.viewAll}
              </Button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-slate-500 uppercase bg-slate-50 dark:bg-slate-900/50 border-b border-slate-100 dark:border-slate-800">
                  <tr>
                    <th className="px-6 py-3 font-medium">{t.dashboard.table.workflow}</th>
                    <th className="px-6 py-3 font-medium">{t.dashboard.table.status}</th>
                    <th className="px-6 py-3 font-medium">{t.dashboard.table.duration}</th>
                    <th className="px-6 py-3 font-medium">{t.dashboard.table.triggeredBy}</th>
                    <th className="px-6 py-3 font-medium text-right">{t.dashboard.table.time}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {loading && recentRuns.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-6 py-6 text-center text-slate-400">
                        <Loader2 size={16} className="inline animate-spin" />
                      </td>
                    </tr>
                  )}
                  {!loading && recentRuns.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-6 py-6 text-center text-slate-400">
                        {t.dashboard.noRecentActivity}
                      </td>
                    </tr>
                  )}
                  {recentRuns.map((run) => (
                    <tr
                      key={run.id}
                      onClick={() => navigate(`/executions/${run.id}`)}
                      className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
                    >
                      <td className="px-6 py-4 font-medium text-slate-900 dark:text-slate-200">
                        {workflowLabel(run)}
                        <div className="text-[10px] text-slate-400 font-normal font-mono mt-0.5">{run.id.substring(0, 8)}</div>
                      </td>
                      <td className="px-6 py-4">
                        <RunStatus status={run.status} label={t.system.statuses[run.status as keyof typeof t.system.statuses] ?? run.status} />
                      </td>
                      <td className="px-6 py-4 text-slate-500 dark:text-slate-400 font-mono text-xs">
                        {formatDuration(executionDurationMs(run))}
                      </td>
                      <td className="px-6 py-4 text-slate-600 dark:text-slate-300">
                        {run.triggered_by ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                            {run.triggered_by}
                          </span>
                        ) : (
                          <span className="text-slate-400">-</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-right text-slate-500 dark:text-slate-400">
                        {formatTimeAgo(run.started_at, timeLabels)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="space-y-6">
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm p-5">
              <div className="flex items-center justify-between mb-4">
                <h2 className="font-bold text-slate-800 dark:text-slate-100 flex items-center">
                  <Server size={18} className="mr-2 text-slate-400" />
                  {t.dashboard.systemHealth}
                </h2>
                <Button variant="ghost" size="sm" onClick={() => navigate('/monitoring')}>
                  {t.dashboard.viewAll}
                </Button>
              </div>
              <HealthPanel health={health} />
            </div>

            <div className="bg-gradient-to-br from-blue-600 to-indigo-700 rounded-xl shadow-lg p-5 text-white">
              <h3 className="font-bold text-lg mb-2">{t.dashboard.quickActions}</h3>
              <p className="text-blue-100 text-sm mb-4">{t.dashboard.quickDesc}</p>
              <Button onClick={handleCreateNew} className="w-full bg-white text-blue-600 font-semibold hover:bg-blue-50 shadow-sm">
                {t.dashboard.createWorkflow}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

const RunStatus: React.FC<{ status: string; label: string }> = ({ status, label }) => {
  const style =
    status === 'completed'
      ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30'
      : status === 'failed' || status === 'timeout'
        ? 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-900/30'
        : status === 'running' || status === 'pending'
          ? 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-900/30'
          : 'bg-slate-50 text-slate-600 border-slate-200 dark:bg-slate-800/50 dark:text-slate-400 dark:border-slate-700';
  const icon =
    status === 'completed' ? (
      <CheckCircle2 size={12} className="mr-1.5" />
    ) : status === 'running' || status === 'pending' ? (
      <Loader2 size={12} className="mr-1.5 animate-spin" />
    ) : status === 'cancelled' ? (
      <XCircle size={12} className="mr-1.5" />
    ) : (
      <AlertCircle size={12} className="mr-1.5" />
    );
  return (
    <span className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium border ${style}`}>
      {icon}
      {label}
    </span>
  );
};
