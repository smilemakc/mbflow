import React from 'react';

const METHOD_STYLES: Record<string, string> = {
    GET: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300',
    POST: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
    PUT: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
    PATCH: 'bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300',
    DELETE: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
};

const DEFAULT_STYLE = 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300';

interface MethodBadgeProps {
    method: string;
    className?: string;
}

export const MethodBadge: React.FC<MethodBadgeProps> = ({method, className = ''}) => {
    const upper = method.toUpperCase();
    return (
        <span
            className={`inline-flex shrink-0 items-center justify-center min-w-[3.25rem] px-1.5 py-0.5 rounded text-[10px] font-bold font-mono tracking-wide ${METHOD_STYLES[upper] ?? DEFAULT_STYLE} ${className}`}
        >
            {upper}
        </span>
    );
};
