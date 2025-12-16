/**
 * CreateStorageModal component
 * Single Responsibility: Modal for creating new file storage resource
 */

import React, {useState} from 'react';
import {Button, Modal} from '@/components/ui';
import {useTranslation} from '@/store/translations';

interface CreateStorageModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSubmit: (name: string, description: string) => Promise<boolean>;
}

export const CreateStorageModal: React.FC<CreateStorageModalProps> = (
    {
        isOpen,
        onClose,
        onSubmit,
    }) => {
    const t = useTranslation();
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
        if (!name.trim()) return;

        setLoading(true);
        const success = await onSubmit(name.trim(), description.trim());
        setLoading(false);

        if (success) {
            setName('');
            setDescription('');
            onClose();
        }
    };

    const handleClose = () => {
        if (!loading) {
            setName('');
            setDescription('');
            onClose();
        }
    };

    return (
        <Modal
            isOpen={isOpen}
            onClose={handleClose}
            title={t.resources.createStorageTitle}
            size="md"
            footer={
                <div className="flex justify-end gap-3">
                    <Button onClick={handleClose} variant="secondary" disabled={loading}>
                        {t.common.cancel}
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        variant="primary"
                        loading={loading}
                        disabled={!name.trim()}
                    >
                        {t.resources.create}
                    </Button>
                </div>
            }
        >
            <div className="space-y-4">
                <FormField
                    label={t.resources.storageName}
                    required
                    value={name}
                    onChange={setName}
                    placeholder={t.resources.storageNamePlaceholder}
                />
                <FormTextArea
                    label={t.resources.description}
                    value={description}
                    onChange={setDescription}
                    placeholder={t.resources.descriptionPlaceholder}
                />
            </div>
        </Modal>
    );
};

interface FormFieldProps {
    label: string;
    required?: boolean;
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
}

const FormField: React.FC<FormFieldProps> = (
    {
        label,
        required,
        value,
        onChange,
        placeholder,
    }) => (
    <div>
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            {label} {required && <span className="text-red-500">*</span>}
        </label>
        <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
        />
    </div>
);

interface FormTextAreaProps {
    label: string;
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    rows?: number;
}

const FormTextArea: React.FC<FormTextAreaProps> = (
    {
        label,
        value,
        onChange,
        placeholder,
        rows = 3,
    }) => (
    <div>
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            {label}
        </label>
        <textarea
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            rows={rows}
            className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
        />
    </div>
);
