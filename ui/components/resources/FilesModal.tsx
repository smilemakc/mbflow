/**
 * FilesModal component
 * Single Responsibility: Modal for managing files in a file storage resource
 */

import React, { useState } from 'react';
import {
  Upload,
  Download,
  Trash2,
  Loader2,
  FolderOpen,
  FileText,
} from 'lucide-react';
import { Button, Modal, ConfirmModal } from '@/components/ui';
import { FileMetadata, FileStorageResource } from '@/services/resources.ts';
import { formatBytes } from '@/utils/formatters.ts';
import { useTranslation } from '@/store/translations';

interface FilesModalProps {
  isOpen: boolean;
  onClose: () => void;
  resource: FileStorageResource | null;
  files: FileMetadata[];
  filesTotal: number;
  loading: boolean;
  onUpload: (file: File) => Promise<boolean>;
  onDelete: (fileId: string, fileName: string) => Promise<boolean>;
  onDownload: (fileId: string) => void;
}

export const FilesModal: React.FC<FilesModalProps> = ({
  isOpen,
  onClose,
  resource,
  files,
  filesTotal,
  loading,
  onUpload,
  onDelete,
  onDownload,
}) => {
  const t = useTranslation();
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [fileToDelete, setFileToDelete] = useState<{id: string, name: string} | null>(null);

  const handleUpload = async () => {
    if (!uploadFile) return;

    setActionLoading(true);
    const success = await onUpload(uploadFile);
    setActionLoading(false);

    if (success) {
      setUploadFile(null);
    }
  };

  const handleDeleteClick = (fileId: string, fileName: string) => {
    setFileToDelete({id: fileId, name: fileName});
  };

  const handleDeleteConfirm = async () => {
    if (!fileToDelete) return;

    setActionLoading(true);
    await onDelete(fileToDelete.id, fileToDelete.name);
    setActionLoading(false);
    setFileToDelete(null);
  };

  const handleClose = () => {
    setUploadFile(null);
    onClose();
  };

  return (
    <>
      <Modal
        isOpen={isOpen}
        onClose={handleClose}
        title={`${t.resources.filesTitle} - ${resource?.name || ''}`}
        size="lg"
      >
        <div className="space-y-4">
          <UploadSection
            uploadFile={uploadFile}
            onFileSelect={setUploadFile}
            onUpload={handleUpload}
            loading={actionLoading}
          />
          <FilesList
            files={files}
            filesTotal={filesTotal}
            loading={loading}
            onDownload={onDownload}
            onDelete={handleDeleteClick}
            actionLoading={actionLoading}
          />
        </div>
      </Modal>

      <ConfirmModal
        isOpen={!!fileToDelete}
        onClose={() => setFileToDelete(null)}
        onConfirm={handleDeleteConfirm}
        title={t.resources.deleteFileTitle}
        message={t.resources.deleteFileMessage.replace('this file', `"${fileToDelete?.name}"`)}
        confirmText={t.common.delete}
        variant="danger"
      />
    </>
  );
};

interface UploadSectionProps {
  uploadFile: File | null;
  onFileSelect: (file: File | null) => void;
  onUpload: () => void;
  loading: boolean;
}

const UploadSection: React.FC<UploadSectionProps> = ({
  uploadFile,
  onFileSelect,
  onUpload,
  loading,
}) => {
  const t = useTranslation();
  return (
  <div className="border border-dashed border-slate-300 dark:border-slate-700 rounded-lg p-4">
    <div className="flex items-center gap-3">
      <input
        type="file"
        id="file-upload"
        className="hidden"
        onChange={(e) => onFileSelect(e.target.files?.[0] || null)}
      />
      <label
        htmlFor="file-upload"
        className="flex-1 flex items-center justify-center px-4 py-3 border border-slate-200 dark:border-slate-700 rounded-lg cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-900 transition-colors"
      >
        <Upload size={16} className="mr-2" />
        <span className="text-sm text-slate-600 dark:text-slate-400">
          {uploadFile ? uploadFile.name : t.resources.chooseFile}
        </span>
      </label>
      {uploadFile && (
        <Button
          onClick={onUpload}
          variant="primary"
          size="sm"
          loading={loading}
          icon={<Upload size={14} />}
        >
          {t.resources.upload}
        </Button>
      )}
    </div>
    {uploadFile && (
      <p className="text-xs text-slate-500 dark:text-slate-400 mt-2">
        {t.resources.fileSize}: {formatBytes(uploadFile.size)}
      </p>
    )}
  </div>
  );
};

interface FilesListProps {
  files: FileMetadata[];
  filesTotal: number;
  loading: boolean;
  onDownload: (fileId: string) => void;
  onDelete: (fileId: string, fileName: string) => void;
  actionLoading: boolean;
}

const FilesList: React.FC<FilesListProps> = ({
  files,
  filesTotal,
  loading,
  onDownload,
  onDelete,
  actionLoading,
}) => {
  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 size={24} className="animate-spin text-blue-600" />
      </div>
    );
  }

  if (files.length === 0) {
    return <EmptyState />;
  }

  return (
    <div className="space-y-2">
      <FilesHeader count={files.length} total={filesTotal} files={files} />
      <div className="space-y-1 max-h-96 overflow-y-auto">
        {files.map((file) => (
          <FileItem
            key={file.id}
            file={file}
            onDownload={onDownload}
            onDelete={onDelete}
            disabled={actionLoading}
          />
        ))}
      </div>
    </div>
  );
};

const EmptyState: React.FC = () => {
  const t = useTranslation();
  return (
  <div className="text-center py-12">
    <FolderOpen size={48} className="mx-auto mb-4 text-slate-300 dark:text-slate-700" />
    <p className="text-slate-500 dark:text-slate-400">{t.resources.noFilesYet}</p>
  </div>
  );
};

interface FilesHeaderProps {
  count: number;
  total: number;
  files: FileMetadata[];
}

const FilesHeader: React.FC<FilesHeaderProps> = ({ count, total, files }) => {
  const t = useTranslation();
  const totalSize = files.reduce((sum, f) => sum + f.size, 0);

  return (
    <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400 px-2">
      <span>{t.resources.showingFiles} {count} {t.resources.ofFiles} {total} {t.resources.totalFiles}</span>
      <span>{t.resources.totalSize}: {formatBytes(totalSize)}</span>
    </div>
  );
};

interface FileItemProps {
  file: FileMetadata;
  onDownload: (fileId: string) => void;
  onDelete: (fileId: string, fileName: string) => void;
  disabled: boolean;
}

const FileItem: React.FC<FileItemProps> = ({
  file,
  onDownload,
  onDelete,
  disabled,
}) => (
  <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800 hover:border-blue-300 dark:hover:border-blue-700 transition-colors">
    <div className="flex items-center gap-3 flex-1 min-w-0">
      <FileText size={16} className="text-blue-600 dark:text-blue-400 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
          {file.name}
        </p>
        <p className="text-xs text-slate-500 dark:text-slate-400">
          {formatBytes(file.size)} • {new Date(file.created_at).toLocaleString()}
        </p>
      </div>
    </div>
    <div className="flex items-center gap-1 shrink-0">
      <Button
        onClick={() => onDownload(file.id)}
        variant="ghost"
        size="sm"
        icon={<Download size={14} />}
        className="text-blue-600 hover:text-blue-700 hover:bg-blue-50 dark:hover:bg-blue-900/20"
      />
      <Button
        onClick={() => onDelete(file.id, file.name)}
        variant="ghost"
        size="sm"
        icon={<Trash2 size={14} />}
        className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
        disabled={disabled}
      />
    </div>
  </div>
);
