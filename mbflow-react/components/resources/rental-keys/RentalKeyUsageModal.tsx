/**
 * RentalKeyUsageModal component
 * Displays usage history for a rental key in a modal.
 */

import React, { useState, useEffect } from 'react';
import {
  X,
  Clock,
  CheckCircle2,
  XCircle,
  ChevronLeft,
  ChevronRight,
  Zap,
  DollarSign,
  Image,
  Music,
  Video,
  FileText,
} from 'lucide-react';
import { Button, Modal } from '@/components/ui';
import {
  RentalKey,
  UsageRecord,
  rentalKeyApi,
  getProviderLabel,
  formatTokenCount,
} from '@/services/rentalKeyService';
import { useTranslation } from '@/store/translations';
import { getErrorMessage } from '@/lib/api';

interface RentalKeyUsageModalProps {
  isOpen: boolean;
  onClose: () => void;
  rentalKey: RentalKey;
}

export const RentalKeyUsageModal: React.FC<RentalKeyUsageModalProps> = ({
  isOpen,
  onClose,
  rentalKey,
}) => {
  const t = useTranslation();
  const [usageRecords, setUsageRecords] = useState<UsageRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const fetchUsage = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await rentalKeyApi.getUsageHistory(rentalKey.id, limit, offset);
      setUsageRecords(response.data.usage || []);
    } catch (error: unknown) {
      setError(getErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      fetchUsage();
    }
  }, [isOpen, offset]);

  const handlePrevPage = () => {
    setOffset(Math.max(0, offset - limit));
  };

  const handleNextPage = () => {
    if (usageRecords.length === limit) {
      setOffset(offset + limit);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="" size="xl">
      <div className="p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-xl font-bold text-slate-900 dark:text-white">
              {t.rentalKeys?.usageHistory || 'Usage History'}
            </h2>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
              {rentalKey.name} - {getProviderLabel(rentalKey.provider)}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
          >
            <X size={20} className="text-slate-500" />
          </button>
        </div>

        {/* Summary stats */}
        <UsageSummary rentalKey={rentalKey} />

        {/* Usage records */}
        <div className="mt-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-600 dark:text-red-400">
              {error}
            </div>
          ) : usageRecords.length === 0 ? (
            <div className="text-center py-12 text-slate-500 dark:text-slate-400">
              {t.rentalKeys?.noUsageRecords || 'No usage records found'}
            </div>
          ) : (
            <>
              <div className="space-y-3">
                {usageRecords.map((record) => (
                  <UsageRecordRow key={record.id} record={record} />
                ))}
              </div>

              {/* Pagination */}
              <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
                <Button
                  onClick={handlePrevPage}
                  disabled={offset === 0}
                  variant="outline"
                  size="sm"
                  icon={<ChevronLeft size={14} />}
                >
                  {t.common?.previous || 'Previous'}
                </Button>
                <span className="text-sm text-slate-500 dark:text-slate-400">
                  {t.rentalKeys?.showingRecords || 'Showing'} {offset + 1} - {offset + usageRecords.length}
                </span>
                <Button
                  onClick={handleNextPage}
                  disabled={usageRecords.length < limit}
                  variant="outline"
                  size="sm"
                  iconPosition="right"
                  icon={<ChevronRight size={14} />}
                >
                  {t.common?.next || 'Next'}
                </Button>
              </div>
            </>
          )}
        </div>
      </div>
    </Modal>
  );
};

interface UsageSummaryProps {
  rentalKey: RentalKey;
}

const UsageSummary: React.FC<UsageSummaryProps> = ({ rentalKey }) => {
  const t = useTranslation();
  const { total_usage } = rentalKey;

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
      {/* Text tokens */}
      <div className="text-center">
        <div className="flex items-center justify-center mb-2">
          <FileText size={16} className="text-blue-500 mr-1" />
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {t.rentalKeys?.textTokens || 'Text'}
          </span>
        </div>
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          {formatTokenCount(total_usage.prompt_tokens + total_usage.completion_tokens)}
        </div>
      </div>

      {/* Image tokens */}
      <div className="text-center">
        <div className="flex items-center justify-center mb-2">
          <Image size={16} className="text-purple-500 mr-1" />
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {t.rentalKeys?.imageTokens || 'Image'}
          </span>
        </div>
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          {formatTokenCount(total_usage.image_input_tokens + total_usage.image_output_tokens)}
        </div>
      </div>

      {/* Audio tokens */}
      <div className="text-center">
        <div className="flex items-center justify-center mb-2">
          <Music size={16} className="text-green-500 mr-1" />
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {t.rentalKeys?.audioTokens || 'Audio'}
          </span>
        </div>
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          {formatTokenCount(total_usage.audio_input_tokens + total_usage.audio_output_tokens)}
        </div>
      </div>

      {/* Video tokens */}
      <div className="text-center">
        <div className="flex items-center justify-center mb-2">
          <Video size={16} className="text-orange-500 mr-1" />
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {t.rentalKeys?.videoTokens || 'Video'}
          </span>
        </div>
        <div className="text-lg font-bold text-slate-900 dark:text-white">
          {formatTokenCount(total_usage.video_input_tokens + total_usage.video_output_tokens)}
        </div>
      </div>
    </div>
  );
};

interface UsageRecordRowProps {
  record: UsageRecord;
}

const UsageRecordRow: React.FC<UsageRecordRowProps> = ({ record }) => {
  const isSuccess = record.status === 'success';
  const formattedDate = new Date(record.created_at).toLocaleString();

  return (
    <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg">
      <div className="flex items-center space-x-3">
        {isSuccess ? (
          <CheckCircle2 size={16} className="text-green-500" />
        ) : (
          <XCircle size={16} className="text-red-500" />
        )}
        <div>
          <div className="font-medium text-slate-900 dark:text-white text-sm">
            {record.model}
          </div>
          <div className="text-xs text-slate-500 dark:text-slate-400 flex items-center">
            <Clock size={10} className="mr-1" />
            {formattedDate}
            {record.response_time_ms && (
              <span className="ml-2">({record.response_time_ms}ms)</span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center space-x-4 text-sm">
        <div className="flex items-center text-slate-600 dark:text-slate-400">
          <Zap size={14} className="mr-1" />
          {formatTokenCount(record.usage.total)}
        </div>
        <div className="flex items-center text-slate-600 dark:text-slate-400">
          <DollarSign size={14} className="mr-1" />
          {record.estimated_cost.toFixed(4)}
        </div>
      </div>
    </div>
  );
};

export default RentalKeyUsageModal;
