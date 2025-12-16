/**
 * TransactionHistory component
 * Single Responsibility: Displays transaction history table with filters
 */

import React from 'react';
import {CreditCard, FileText, TrendingUp} from 'lucide-react';
import {Button} from '@/components/ui';
import {Transaction} from '@/services/resources.ts';
import {formatDate} from '@/utils/formatters.ts';

interface TransactionHistoryProps {
    transactions: Transaction[];
    total: number;
    onDepositClick: () => void;
}

export const TransactionHistory: React.FC<TransactionHistoryProps> = (
    {
        transactions,
        total,
        onDepositClick,
    }) => {
    return (
        <section>
            <SectionHeader onDepositClick={onDepositClick}/>
            <TransactionTable transactions={transactions} total={total}/>
        </section>
    );
};

interface SectionHeaderProps {
    onDepositClick: () => void;
}

const SectionHeader: React.FC<SectionHeaderProps> = ({onDepositClick}) => (
    <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Billing & Transactions
        </h2>
        <Button
            onClick={onDepositClick}
            variant="outline"
            size="sm"
            icon={<CreditCard size={16}/>}
        >
            Deposit Funds
        </Button>
    </div>
);

interface TransactionTableProps {
    transactions: Transaction[];
    total: number;
}

const TransactionTable: React.FC<TransactionTableProps> = ({transactions, total}) => (
    <div
        className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">
        <TableHeader count={transactions.length} total={total}/>
        {transactions.length === 0 ? (
            <EmptyState/>
        ) : (
            <TableBody transactions={transactions}/>
        )}
    </div>
);

interface TableHeaderProps {
    count: number;
    total: number;
}

const TableHeader: React.FC<TableHeaderProps> = ({count, total}) => (
    <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800">
        <h3 className="font-bold text-slate-900 dark:text-white flex items-center">
            <TrendingUp size={18} className="mr-2"/>
            Transaction History
        </h3>
        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Showing {count} of {total} total transactions
        </p>
    </div>
);

const EmptyState: React.FC = () => (
    <div className="p-12 text-center">
        <FileText size={48} className="mx-auto mb-4 text-slate-300 dark:text-slate-700"/>
        <p className="text-slate-500 dark:text-slate-400">No transactions yet</p>
    </div>
);

interface TableBodyProps {
    transactions: Transaction[];
}

const TableBody: React.FC<TableBodyProps> = ({transactions}) => (
    <div className="overflow-x-auto">
        <table className="w-full">
            <thead className="bg-slate-50 dark:bg-slate-900/50">
            <tr>
                <TableHeaderCell>Type</TableHeaderCell>
                <TableHeaderCell>Description</TableHeaderCell>
                <TableHeaderCell align="right">Amount</TableHeaderCell>
                <TableHeaderCell align="center">Status</TableHeaderCell>
                <TableHeaderCell align="right">Date</TableHeaderCell>
            </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
            {transactions.map((tx) => (
                <TransactionRow key={tx.id} transaction={tx}/>
            ))}
            </tbody>
        </table>
    </div>
);

interface TableHeaderCellProps {
    children: React.ReactNode;
    align?: 'left' | 'center' | 'right';
}

const TableHeaderCell: React.FC<TableHeaderCellProps> = ({children, align = 'left'}) => (
    <th
        className={`px-6 py-3 text-${align} text-xs font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider`}
    >
        {children}
    </th>
);

interface TransactionRowProps {
    transaction: Transaction;
}

const TransactionRow: React.FC<TransactionRowProps> = ({transaction: tx}) => {
    const isPositive = tx.type === 'deposit' || tx.type === 'refund';

    return (
        <tr className="hover:bg-slate-50 dark:hover:bg-slate-900/50 transition-colors">
            <td className="px-6 py-4 whitespace-nowrap">
                <TypeBadge type={tx.type}/>
            </td>
            <td className="px-6 py-4 text-sm text-slate-900 dark:text-white">
                {tx.description || '-'}
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right">
                <AmountCell amount={tx.amount} currency={tx.currency} isPositive={isPositive}/>
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-center">
                <StatusBadge status={tx.status}/>
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-slate-500 dark:text-slate-400">
                {formatDate(tx.created_at)}
            </td>
        </tr>
    );
};

interface TypeBadgeProps {
    type: string;
}

const TypeBadge: React.FC<TypeBadgeProps> = ({type}) => (
    <span
        className="inline-flex items-center px-2 py-1 rounded text-xs font-medium bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400">
    {type}
  </span>
);

interface AmountCellProps {
    amount: number;
    currency: string;
    isPositive: boolean;
}

const AmountCell: React.FC<AmountCellProps> = ({amount, currency, isPositive}) => (
    <span
        className={`text-sm font-bold ${
            isPositive
                ? 'text-green-600 dark:text-green-400'
                : 'text-red-600 dark:text-red-400'
        }`}
    >
    {isPositive ? '+' : '-'}
        {Math.abs(amount).toFixed(2)} {currency}
  </span>
);

interface StatusBadgeProps {
    status: string;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({status}) => {
    const getStatusStyles = (status: string): string => {
        switch (status) {
            case 'completed':
                return 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400';
            case 'pending':
                return 'bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-400';
            default:
                return 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400';
        }
    };

    return (
        <span
            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${getStatusStyles(status)}`}
        >
      {status}
    </span>
    );
};
