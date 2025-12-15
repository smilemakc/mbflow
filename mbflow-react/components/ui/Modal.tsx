import React, {useEffect} from 'react';
import {createPortal} from 'react-dom';
import {X} from 'lucide-react';
import {Button} from './Button';

export interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
    closeOnBackdrop?: boolean;
    closeOnEscape?: boolean;
    showCloseButton?: boolean;
    children: React.ReactNode;
    footer?: React.ReactNode;
}

const sizeClasses = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    full: 'max-w-4xl',
};

export const Modal: React.FC<ModalProps> = ({
                                                isOpen,
                                                onClose,
                                                title,
                                                size = 'md',
                                                closeOnBackdrop = true,
                                                closeOnEscape = true,
                                                showCloseButton = true,
                                                children,
                                                footer,
                                            }) => {
    useEffect(() => {
        if (!isOpen || !closeOnEscape) return;

        const handleEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                onClose();
            }
        };

        document.addEventListener('keydown', handleEscape);
        return () => document.removeEventListener('keydown', handleEscape);
    }, [isOpen, closeOnEscape, onClose]);

    useEffect(() => {
        if (isOpen) {
            document.body.style.overflow = 'hidden';
        } else {
            document.body.style.overflow = '';
        }

        return () => {
            document.body.style.overflow = '';
        };
    }, [isOpen]);

    if (!isOpen) return null;

    const handleBackdropClick = (event: React.MouseEvent<HTMLDivElement>) => {
        if (closeOnBackdrop && event.target === event.currentTarget) {
            onClose();
        }
    };

    const modalContent = (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            {/* Backdrop */}
            <div
                className="fixed inset-0 bg-black/30 animate-in fade-in duration-300"
                onClick={handleBackdropClick}
                aria-hidden="true"
            />

            {/* Modal Panel */}
            <div
                className={`
          relative w-full overflow-hidden rounded-lg bg-white shadow-xl
          animate-in fade-in zoom-in-95 duration-300
          ${sizeClasses[size]}
        `}
                role="dialog"
                aria-modal="true"
                aria-labelledby={title ? 'modal-title' : undefined}
            >
                {/* Header */}
                {(title || showCloseButton) && (
                    <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
                        {title && (
                            <h2
                                id="modal-title"
                                className="text-lg font-semibold text-gray-900"
                            >
                                {title}
                            </h2>
                        )}
                        {showCloseButton && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={onClose}
                                aria-label="Close modal"
                                icon={<X className="h-5 w-5" />}
                            />
                        )}
                    </div>
                )}

                {/* Content */}
                <div className="px-6 py-4">{children}</div>

                {/* Footer */}
                {footer && (
                    <div className="border-t border-gray-200 bg-gray-50 px-6 py-4">
                        {footer}
                    </div>
                )}
            </div>
        </div>
    );

    return createPortal(modalContent, document.body);
};
