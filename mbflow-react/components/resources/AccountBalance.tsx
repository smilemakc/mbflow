/**
 * AccountBalance component
 * Single Responsibility: Displays user's account balance and status
 */

import React from 'react';
import {AlertCircle, CheckCircle2, DollarSign} from 'lucide-react';
import {Account} from '@/services/resources.ts';

interface AccountBalanceProps {
    account: Account;
}

export const AccountBalance: React.FC<AccountBalanceProps> = ({account}) => {
    const isActive = account.status === 'active';

    return (
        <div
            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm min-w-[180px]">
            <div className="text-sm text-slate-500 dark:text-slate-400 mb-1">
                Account Balance
            </div>
            <div className="text-2xl font-bold text-slate-900 dark:text-white flex items-center">
                <DollarSign size={20} className="mr-1"/>
                {account.balance.toFixed(2)} {account.currency}
            </div>
            <div className="mt-2">
                <StatusBadge status={account.status} isActive={isActive}/>
            </div>
        </div>
    );
};

interface StatusBadgeProps {
    status: string;
    isActive: boolean;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({status, isActive}) => (
    <span
        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
            isActive
                ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400'
                : 'bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-400'
        }`}
    >
    {isActive ? (
        <CheckCircle2 size={10} className="mr-1"/>
    ) : (
        <AlertCircle size={10} className="mr-1"/>
    )}
        {status.charAt(0).toUpperCase() + status.slice(1)}
  </span>
);
