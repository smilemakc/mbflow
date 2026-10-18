import React, {useRef, useState} from 'react';
import {AlertTriangle, FileCode2, Loader2, UploadCloud} from 'lucide-react';
import {Button, Modal} from '@/components/ui';
import {FormField} from '@/components/ui/form';
import {configStyles} from '@/styles/configStyles';
import {useTranslation} from '@/store/translations';
import {getErrorMessage} from '@/lib/api';
import {toast} from '@/lib/toast';
import {apiClientService, ApiClient, ApiClientAuth, ApiSpecPreview} from '@/services/apiClientService';
import {AuthSettingsForm} from './AuthSettingsForm';
import {BaseUrlField} from './BaseUrlField';
import {format} from './schemaUtils';

interface ImportApiClientModalProps {
    isOpen: boolean;
    onClose: () => void;
    onImported: (client: ApiClient) => void;
}

const ACCEPTED_FILES = '.json,.yaml,.yml,application/json,application/yaml,application/x-yaml,text/yaml';

export const ImportApiClientModal: React.FC<ImportApiClientModalProps> = ({isOpen, onClose, onImported}) => {
    const t = useTranslation();
    const inputRef = useRef<HTMLInputElement>(null);
    const [file, setFile] = useState<File | null>(null);
    const [preview, setPreview] = useState<ApiSpecPreview | null>(null);
    const [previewError, setPreviewError] = useState<string | null>(null);
    const [isParsing, setIsParsing] = useState(false);
    const [isImporting, setIsImporting] = useState(false);
    const [isDragging, setIsDragging] = useState(false);
    const [name, setName] = useState('');
    const [baseUrl, setBaseUrl] = useState('');
    const [auth, setAuth] = useState<ApiClientAuth>({type: 'none'});

    const reset = () => {
        setFile(null);
        setPreview(null);
        setPreviewError(null);
        setName('');
        setBaseUrl('');
        setAuth({type: 'none'});
    };

    const close = () => {
        reset();
        onClose();
    };

    const selectFile = async (selected: File) => {
        setFile(selected);
        setPreview(null);
        setPreviewError(null);
        setIsParsing(true);
        try {
            const response = await apiClientService.preview(selected);
            const spec = response.data;
            setPreview(spec);
            setName(spec.title);
            setBaseUrl(spec.base_url ?? '');
            setAuth(spec.suggested_auth ?? {type: 'none'});
        } catch (error) {
            setPreviewError(getErrorMessage(error));
        } finally {
            setIsParsing(false);
        }
    };

    const handleDrop = (event: React.DragEvent) => {
        event.preventDefault();
        setIsDragging(false);
        const dropped = event.dataTransfer.files?.[0];
        if (dropped) void selectFile(dropped);
    };

    const handleImport = async () => {
        if (!file || !preview) return;
        setIsImporting(true);
        try {
            const response = await apiClientService.create(file, {
                name: name.trim() || preview.title,
                base_url: baseUrl.trim() || undefined,
                auth,
            });
            toast.success(t.apiClients.imported, format(t.apiClients.importedMessage, {count: response.data.operation_count}));
            onImported(response.data);
            reset();
            onClose();
        } catch (error) {
            toast.error(t.common.error, getErrorMessage(error));
        } finally {
            setIsImporting(false);
        }
    };

    const footer = (
        <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={close}>{t.common.cancel}</Button>
            <Button
                onClick={handleImport}
                loading={isImporting}
                disabled={!preview || isParsing || !name.trim()}
            >
                {t.apiClients.importButton}
            </Button>
        </div>
    );

    return (
        <Modal isOpen={isOpen} onClose={close} title={t.apiClients.importTitle} size="full" footer={footer}>
            <div className="space-y-5">
                <input
                    ref={inputRef}
                    type="file"
                    accept={ACCEPTED_FILES}
                    className="hidden"
                    onChange={(e) => {
                        const selected = e.target.files?.[0];
                        if (selected) void selectFile(selected);
                        e.target.value = '';
                    }}
                />

                {!preview && (
                    <button
                        type="button"
                        onClick={() => inputRef.current?.click()}
                        onDragOver={(e) => {
                            e.preventDefault();
                            setIsDragging(true);
                        }}
                        onDragLeave={() => setIsDragging(false)}
                        onDrop={handleDrop}
                        disabled={isParsing}
                        className={`w-full flex flex-col items-center justify-center gap-3 px-6 py-12 rounded-xl border-2 border-dashed transition-colors ${
                            isDragging
                                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                                : 'border-slate-300 dark:border-slate-700 hover:border-blue-400 bg-slate-50 dark:bg-slate-900/50'
                        }`}
                    >
                        {isParsing ? (
                            <Loader2 size={36} className="text-blue-500 animate-spin"/>
                        ) : (
                            <UploadCloud size={36} className="text-slate-400"/>
                        )}
                        <span className="text-sm font-medium text-slate-700 dark:text-slate-200">
                            {isParsing ? t.apiClients.parsing : t.apiClients.dropFile}
                        </span>
                        <span className="text-xs text-slate-500 dark:text-slate-400">{t.apiClients.fileTypes}</span>
                        {file && !isParsing && (
                            <span className="text-xs font-mono text-slate-500">{file.name}</span>
                        )}
                    </button>
                )}

                {previewError && (
                    <div className={`${configStyles.infoBox.error} flex gap-2 text-sm text-red-700 dark:text-red-300`}>
                        <AlertTriangle size={16} className="shrink-0 mt-0.5"/>
                        <span className="break-words min-w-0">{previewError}</span>
                    </div>
                )}

                {preview && (
                    <>
                        <div className="flex items-start gap-3 p-4 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
                            <div className="p-2 rounded-lg bg-indigo-50 dark:bg-indigo-900/30">
                                <FileCode2 size={20} className="text-indigo-500"/>
                            </div>
                            <div className="flex-1 min-w-0">
                                <div className="flex flex-wrap items-baseline gap-x-2">
                                    <h3 className="font-semibold text-slate-900 dark:text-slate-100 truncate">{preview.title}</h3>
                                    {preview.api_version && (
                                        <span className="text-xs text-slate-500">v{preview.api_version}</span>
                                    )}
                                </div>
                                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                    {preview.spec_version === '2.0' ? 'Swagger 2.0' : `OpenAPI ${preview.spec_version}`}
                                    {' · '}
                                    {format(t.apiClients.operationsCount, {count: preview.operation_count})}
                                    {file && <span className="font-mono">{' · '}{file.name}</span>}
                                </p>
                            </div>
                            <Button variant="ghost" size="sm" onClick={() => inputRef.current?.click()}>
                                {t.apiClients.chooseAnother}
                            </Button>
                        </div>

                        {preview.warnings.length > 0 && (
                            <div className={`${configStyles.infoBox.warning} space-y-1`}>
                                <p className="text-xs font-semibold text-amber-800 dark:text-amber-300">{t.apiClients.warnings}</p>
                                <ul className="list-disc pl-4 space-y-0.5 text-xs text-amber-800 dark:text-amber-300">
                                    {preview.warnings.map((w, i) => (
                                        <li key={i} className="break-words">{w.message}</li>
                                    ))}
                                </ul>
                            </div>
                        )}

                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="space-y-4">
                                <FormField label={t.apiClients.name} required>
                                    <input
                                        type="text"
                                        value={name}
                                        onChange={(e) => setName(e.target.value)}
                                        className={configStyles.input}
                                        maxLength={255}
                                    />
                                </FormField>
                                <BaseUrlField value={baseUrl} onChange={setBaseUrl} servers={preview.servers}/>
                            </div>
                            <div className={configStyles.section}>
                                <h4 className={configStyles.sectionTitle}>{t.apiClients.auth}</h4>
                                <AuthSettingsForm value={auth} onChange={setAuth} securitySchemes={preview.security_schemes}/>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </Modal>
    );
};
