import React from 'react';
import type { SystemHealth } from '@/services/systemService';
import { useTranslation } from '@/store/translations';

type Tone = 'ok' | 'bad' | 'unknown';

const DOT: Record<Tone, string> = {
  ok: 'bg-green-500',
  bad: 'bg-red-500',
  unknown: 'bg-slate-300 dark:bg-slate-600',
};

const TEXT: Record<Tone, string> = {
  ok: 'text-green-600 dark:text-green-400',
  bad: 'text-red-600 dark:text-red-400',
  unknown: 'text-slate-400',
};

interface Row {
  key: string;
  label: string;
  tone: Tone;
  status: string;
}

export interface HealthPanelProps {
  health: SystemHealth | null;
}

/** Service status rows built from GET /health and GET /ready. */
export const HealthPanel: React.FC<HealthPanelProps> = ({ health }) => {
  const t = useTranslation();
  const s = t.system;

  if (!health) {
    return <p className="text-sm text-slate-400">{s.checking}</p>;
  }

  const labels: Record<string, string> = { database: s.database, redis: s.redis };
  const rows: Row[] = [
    {
      key: 'api',
      label: s.api,
      tone: health.apiUp ? 'ok' : 'bad',
      status: health.apiUp
        ? health.status === 'shutting_down'
          ? s.shuttingDown
          : s.operational
        : s.unreachable,
    },
  ];

  for (const [name, state] of Object.entries(health.checks)) {
    rows.push({
      key: name,
      label: labels[name] ?? name,
      tone: state === 'ok' ? 'ok' : 'bad',
      status: state === 'ok' ? s.operational : s.unavailable,
    });
  }

  // Dependency checks are unknown if the readiness probe did not answer.
  if (health.ready === null) {
    rows.push({ key: 'readiness', label: s.database, tone: 'unknown', status: s.unreachable });
  }

  return (
    <ul className="space-y-3" aria-label={s.health}>
      {rows.map((row) => (
        <li key={row.key} className="flex items-center justify-between text-sm" data-testid={`health-${row.key}`}>
          <span className="flex items-center">
            <span className={`w-2 h-2 rounded-full mr-2 ${DOT[row.tone]}`} aria-hidden="true" />
            <span className="text-slate-700 dark:text-slate-300">{row.label}</span>
          </span>
          <span className={`text-xs font-medium ${TEXT[row.tone]}`}>{row.status}</span>
        </li>
      ))}
    </ul>
  );
};

export default HealthPanel;
