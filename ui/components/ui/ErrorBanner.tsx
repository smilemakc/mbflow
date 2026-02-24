import React, { useState } from 'react';
import { XCircle, Copy, Check } from 'lucide-react';

interface ErrorBannerProps {
  title?: string;
  message: string;
  showCopy?: boolean;
}

export const ErrorBanner: React.FC<ErrorBannerProps> = ({
  title = 'Error',
  message,
  showCopy = true,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(message);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-900/30 rounded-xl p-5">
      <div className="flex items-start gap-4">
        <XCircle className="text-red-600 dark:text-red-400 mt-0.5 shrink-0" size={24} />
        <div className="flex-1 min-w-0">
          <h3 className="text-base font-semibold text-red-900 dark:text-red-300 mb-2">
            {title}
          </h3>
          <pre className="text-sm text-red-800 dark:text-red-400 font-mono whitespace-pre-wrap break-words bg-red-100 dark:bg-red-900/30 rounded-lg p-3">
            {message}
          </pre>
        </div>
        {showCopy && (
          <button
            onClick={handleCopy}
            className="shrink-0 p-2 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-lg transition-colors"
            title="Copy to clipboard"
          >
            {copied ? (
              <Check size={16} className="text-green-600 dark:text-green-400" />
            ) : (
              <Copy size={16} className="text-red-600 dark:text-red-400" />
            )}
          </button>
        )}
      </div>
    </div>
  );
};
