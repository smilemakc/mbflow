import React, { useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { Button } from '../ui';

interface CopyButtonProps {
  text: string;
  label?: string;
}

export const CopyButton: React.FC<CopyButtonProps> = ({ text, label }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={handleCopy}
      title={label || 'Copy'}
      icon={copied ? <Check size={12} className="text-green-500" /> : <Copy size={12} />}
    >
      {copied ? (
        <span className="text-green-600 dark:text-green-400">Copied!</span>
      ) : (
        <span>{label || 'Copy'}</span>
      )}
    </Button>
  );
};
