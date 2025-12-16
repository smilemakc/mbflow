/**
 * DepositModal component
 * Single Responsibility: Modal for depositing funds to account
 */

import React, { useState } from 'react';
import { DollarSign, CreditCard } from 'lucide-react';
import { Button, Modal } from '@/components/ui';

interface DepositModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (amount: number) => Promise<boolean>;
}

export const DepositModal: React.FC<DepositModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
}) => {
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);

  const parsedAmount = parseFloat(amount);
  const isValidAmount = !isNaN(parsedAmount) && parsedAmount > 0;

  const handleSubmit = async () => {
    if (!isValidAmount) return;

    setLoading(true);
    const success = await onSubmit(parsedAmount);
    setLoading(false);

    if (success) {
      setAmount('');
      onClose();
    }
  };

  const handleClose = () => {
    if (!loading) {
      setAmount('');
      onClose();
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title="Deposit Funds"
      size="md"
      footer={
        <div className="flex justify-end gap-3">
          <Button onClick={handleClose} variant="secondary" disabled={loading}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="primary"
            loading={loading}
            disabled={!isValidAmount}
            icon={<CreditCard size={16} />}
          >
            Deposit
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <AmountInput value={amount} onChange={setAmount} />
      </div>
    </Modal>
  );
};

interface AmountInputProps {
  value: string;
  onChange: (value: string) => void;
}

const AmountInput: React.FC<AmountInputProps> = ({ value, onChange }) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
      Amount <span className="text-red-500">*</span>
    </label>
    <div className="relative">
      <DollarSign
        size={18}
        className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
      />
      <input
        type="number"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="0.00"
        min="0"
        step="0.01"
        className="w-full pl-10 pr-4 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
      />
    </div>
    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
      Enter the amount you want to add to your account balance
    </p>
  </div>
);
