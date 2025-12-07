import React from 'react';
import {useToast} from '../../hooks/useToast';
import {Toast} from './Toast';

export function ToastContainer() {
    const {toasts, dismissToast} = useToast();

    if (toasts.length === 0) {
        return null;
    }

    return (
        <div className="pointer-events-none fixed right-4 top-4 z-50 space-y-2">
            {toasts.map((toast) => (
                <Toast key={toast.id} toast={toast} onClose={dismissToast}/>
            ))}
        </div>
    );
}
