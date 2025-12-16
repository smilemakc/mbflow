import React from 'react';
import {AlertTriangle, Trash2, Info} from 'lucide-react';
import {Modal} from './Modal';
import {Button} from './Button';

export type ConfirmVariant = 'danger' | 'warning' | 'info';

export interface ConfirmModalProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    variant?: ConfirmVariant;
    isLoading?: boolean;
}

const variantConfig = {
    danger: {
        icon: Trash2,
        iconBg: 'bg-red-100 dark:bg-red-900/30',
        iconColor: 'text-red-600 dark:text-red-400',
        buttonVariant: 'danger' as const,
    },
    warning: {
        icon: AlertTriangle,
        iconBg: 'bg-yellow-100 dark:bg-yellow-900/30',
        iconColor: 'text-yellow-600 dark:text-yellow-400',
        buttonVariant: 'primary' as const,
    },
    info: {
        icon: Info,
        iconBg: 'bg-blue-100 dark:bg-blue-900/30',
        iconColor: 'text-blue-600 dark:text-blue-400',
        buttonVariant: 'primary' as const,
    },
};

export const ConfirmModal: React.FC<ConfirmModalProps> = ({
    isOpen,
    onClose,
    onConfirm,
    title,
    message,
    confirmText = 'Confirm',
    cancelText = 'Cancel',
    variant = 'danger',
    isLoading = false,
}) => {
    const config = variantConfig[variant];
    const Icon = config.icon;

    const handleConfirm = () => {
        onConfirm();
        onClose();
    };

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            size="sm"
            showCloseButton={false}
            closeOnEscape={!isLoading}
            closeOnBackdrop={!isLoading}
        >
            <div className="flex flex-col items-center text-center">
                <div className={`p-3 rounded-full ${config.iconBg} mb-4`}>
                    <Icon size={24} className={config.iconColor} />
                </div>

                <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                    {title}
                </h3>

                <p className="text-sm text-slate-600 dark:text-slate-400 mb-6">
                    {message}
                </p>

                <div className="flex gap-3 w-full">
                    <Button
                        variant="outline"
                        onClick={onClose}
                        disabled={isLoading}
                        className="flex-1"
                    >
                        {cancelText}
                    </Button>
                    <Button
                        variant={config.buttonVariant}
                        onClick={handleConfirm}
                        disabled={isLoading}
                        className="flex-1"
                    >
                        {confirmText}
                    </Button>
                </div>
            </div>
        </Modal>
    );
};
