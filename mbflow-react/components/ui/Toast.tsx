import React, {useEffect, useState} from 'react';
import {AlertTriangle, CheckCircle, Info, X, XCircle} from 'lucide-react';
import type {Toast as ToastType} from '@/hooks/useToast';

interface ToastProps {
    toast: ToastType;
    onClose: (id: string) => void;
}

const iconMap = {
    success: CheckCircle,
    error: XCircle,
    warning: AlertTriangle,
    info: Info,
};

const colorClasses = {
    success: 'bg-green-50 border-green-200 text-green-800',
    error: 'bg-red-50 border-red-200 text-red-800',
    warning: 'bg-yellow-50 border-yellow-200 text-yellow-800',
    info: 'bg-blue-50 border-blue-200 text-blue-800',
};

const iconColorClasses = {
    success: 'text-green-600',
    error: 'text-red-600',
    warning: 'text-yellow-600',
    info: 'text-blue-600',
};

const progressBarClasses = {
    success: 'bg-green-600',
    error: 'bg-red-600',
    warning: 'bg-yellow-600',
    info: 'bg-blue-600',
};

export function Toast({toast, onClose}: ToastProps) {
    const [progress, setProgress] = useState(100);
    const Icon = iconMap[toast.type];

    useEffect(() => {
        if (toast.duration <= 0) return;

        const startTime = Date.now();
        const interval = 16;

        const timer = setInterval(() => {
            const elapsed = Date.now() - startTime;
            const remaining = Math.max(0, 100 - (elapsed / toast.duration) * 100);
            setProgress(remaining);

            if (remaining <= 0) {
                clearInterval(timer);
            }
        }, interval);

        return () => clearInterval(timer);
    }, [toast.duration]);

    return (
        <div
            className={`
        pointer-events-auto w-full max-w-sm rounded-lg border shadow-lg p-4
        ${colorClasses[toast.type]}
        animate-slide-in
      `}
            role="alert"
        >
            <div className="flex items-start gap-3">
                <Icon className={`mt-0.5 h-5 w-5 flex-shrink-0 ${iconColorClasses[toast.type]}`}/>

                <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold">{toast.title}</p>
                    {toast.message && (
                        <p className="mt-1 text-sm opacity-90">{toast.message}</p>
                    )}
                </div>

                <button
                    type="button"
                    className="ml-auto flex-shrink-0 opacity-70 transition-opacity hover:opacity-100"
                    onClick={() => onClose(toast.id)}
                    aria-label="Close notification"
                >
                    <X className="h-5 w-5"/>
                </button>
            </div>

            {toast.duration > 0 && (
                <div className="mt-3 h-1 overflow-hidden rounded-full bg-black/10">
                    <div
                        className={`h-full transition-all ease-linear ${progressBarClasses[toast.type]}`}
                        style={{
                            width: `${progress}%`,
                            transitionDuration: '16ms',
                        }}
                    />
                </div>
            )}
        </div>
    );
}
