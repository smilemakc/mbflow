import React from 'react';
import {Loader2} from 'lucide-react';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline';
    size?: 'sm' | 'md' | 'lg';
    loading?: boolean;
    icon?: React.ReactNode;
    iconPosition?: 'left' | 'right';
    textPosition?: 'left' | 'right' | 'center';
    fullWidth?: boolean;
    children?: React.ReactNode;
}

const cn = (...classes: (string | undefined | null | false)[]): string => {
    return classes.filter(Boolean).join(' ');
};

const variantStyles: Record<string, string> = {
    primary: 'bg-blue-600 hover:bg-blue-700 text-white disabled:bg-blue-400',
    secondary: 'bg-gray-200 hover:bg-gray-300 text-gray-800 disabled:bg-gray-100',
    danger: 'bg-red-600 hover:bg-red-700 text-white disabled:bg-red-400',
    ghost: 'hover:bg-gray-100 text-gray-700 dark:hover:bg-gray-800 dark:text-gray-200 disabled:hover:bg-transparent',
    outline: 'border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-200 disabled:hover:bg-transparent',
};

const sizeStyles: Record<string, string> = {
    sm: 'px-2 py-1 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg',
};

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    (
        {
            variant = 'primary',
            size = 'md',
            loading = false,
            icon,
            iconPosition = 'left',
            textPosition = 'center',
            fullWidth = false,
            disabled = false,
            children,
            className,
            ...props
        },
        ref
    ) => {
        const isDisabled = disabled || loading;

        const baseStyles =
            'inline-flex items-center justify-center rounded-lg font-medium transition-all focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-slate-900 disabled:cursor-not-allowed disabled:opacity-50';

        const widthStyles = fullWidth ? 'w-full' : '';

        const variantClass = variantStyles[variant] || variantStyles.primary;
        const sizeClass = sizeStyles[size] || sizeStyles.md;

        const focusRingColor = {
            primary: 'focus:ring-blue-500',
            secondary: 'focus:ring-gray-400',
            danger: 'focus:ring-red-500',
            ghost: 'focus:ring-gray-400',
            outline: 'focus:ring-gray-400',
        }[variant];

        const buttonClass = cn(
            baseStyles,
            variantClass,
            sizeClass,
            widthStyles,
            focusRingColor,
            className
        );

        const renderContent = () => {
            if (loading) {
                return (
                    <div className="flex items-center gap-2">
                        <Loader2 size={16} className="animate-spin"/>
                        {children}
                    </div>
                );
            }

            if (!icon) {
                return children;
            }

            if (iconPosition === 'left') {
                return (
                    <div className="flex items-center gap-2">
                        <span className="flex-shrink-0">{icon}</span>
                        {children}
                    </div>
                );
            }

            return (
                <div className="flex items-center gap-2">
                    {children}
                    <span className="flex-shrink-0">{icon}</span>
                </div>
            );
        };

        return (
            <button
                ref={ref}
                disabled={isDisabled}
                className={buttonClass}
                {...props}
            >
                {renderContent()}
            </button>
        );
    }
);

Button.displayName = 'Button';
