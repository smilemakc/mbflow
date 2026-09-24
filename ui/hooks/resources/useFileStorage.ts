/**
 * Hook for managing file storage operations
 * Single Responsibility: Handles file upload, download, delete, and listing
 */

import { useState, useCallback } from 'react';
import { resourcesApi, FileMetadata, FileStorageResource } from '@/services/resources.ts';
import { toast } from '@/lib/toast.ts';
import { getErrorMessage } from '@/lib/api';

export interface FileStorageState {
  files: FileMetadata[];
  filesTotal: number;
  filesLoading: boolean;
  selectedResource: FileStorageResource | null;
}

export interface FileStorageActions {
  loadFiles: (resource: FileStorageResource) => Promise<void>;
  uploadFile: (file: File) => Promise<boolean>;
  deleteFile: (fileId: string, fileName: string) => Promise<boolean>;
  downloadFile: (fileId: string) => Promise<void>;
  clearSelection: () => void;
}

export const useFileStorage = (onResourceUpdate?: () => Promise<void>): FileStorageState & FileStorageActions => {
  const [files, setFiles] = useState<FileMetadata[]>([]);
  const [filesTotal, setFilesTotal] = useState(0);
  const [filesLoading, setFilesLoading] = useState(false);
  const [selectedResource, setSelectedResource] = useState<FileStorageResource | null>(null);

  const loadFiles = useCallback(async (resource: FileStorageResource) => {
    setSelectedResource(resource);
    setFilesLoading(true);
    try {
      const response = await resourcesApi.listFiles(resource.id);
      setFiles(response.data.files || []);
      setFilesTotal(response.data.total || 0);
    } catch (error: unknown) {
      console.error('Failed to load files:', error);
      toast.error('Load Failed', getErrorMessage(error));
    } finally {
      setFilesLoading(false);
    }
  }, []);

  const uploadFile = useCallback(async (file: File): Promise<boolean> => {
    if (!selectedResource) {
      toast.error('Error', 'No resource selected.');
      return false;
    }

    try {
      await resourcesApi.uploadFile(selectedResource.id, file);
      toast.success('Success', 'File uploaded successfully.');
      await loadFiles(selectedResource);
      if (onResourceUpdate) {
        await onResourceUpdate();
      }
      return true;
    } catch (error: unknown) {
      console.error('Failed to upload file:', error);
      toast.error('Upload Failed', getErrorMessage(error));
      return false;
    }
  }, [selectedResource, loadFiles, onResourceUpdate]);

  const deleteFile = useCallback(async (fileId: string, fileName: string): Promise<boolean> => {
    if (!selectedResource) {
      toast.error('Error', 'No resource selected.');
      return false;
    }

    try {
      await resourcesApi.deleteFile(selectedResource.id, fileId);
      toast.success('Success', 'File deleted successfully.');
      await loadFiles(selectedResource);
      if (onResourceUpdate) {
        await onResourceUpdate();
      }
      return true;
    } catch (error: unknown) {
      console.error('Failed to delete file:', error);
      toast.error('Delete Failed', getErrorMessage(error));
      return false;
    }
  }, [selectedResource, loadFiles, onResourceUpdate]);

  const downloadFile = useCallback(async (fileId: string) => {
    if (!selectedResource) {
      toast.error('Error', 'No resource selected.');
      return;
    }

    try {
      const fileName = files.find((f) => f.id === fileId)?.name;
      await resourcesApi.downloadFile(selectedResource.id, fileId, fileName);
    } catch (error: unknown) {
      console.error('Failed to download file:', error);
      toast.error('Download Failed', getErrorMessage(error));
    }
  }, [selectedResource, files]);

  const clearSelection = useCallback(() => {
    setSelectedResource(null);
    setFiles([]);
    setFilesTotal(0);
  }, []);

  return {
    files,
    filesTotal,
    filesLoading,
    selectedResource,
    loadFiles,
    uploadFile,
    deleteFile,
    downloadFile,
    clearSelection,
  };
};
