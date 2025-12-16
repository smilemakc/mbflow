import React, {useEffect, useRef} from 'react';
import {useTranslation} from '@/store/translations';
import {Clipboard, Copy, Crosshair, Trash2} from 'lucide-react';

interface ContextMenuProps {
    x: number;
    y: number;
    type: 'node' | 'pane';
    targetId?: string;
    onClose: () => void;
    onAction: (action: string, payload?: any) => void;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
                                                            x,
                                                            y,
                                                            type,
                                                            targetId,
                                                            onClose,
                                                            onAction
                                                        }) => {
    const t = useTranslation();
    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [onClose]);

    const handleAction = (action: string) => {
        onAction(action, targetId);
        onClose();
    };

    return (
        <div
            ref={menuRef}
            className="fixed z-50 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg shadow-xl py-1 min-w-[160px] animate-in fade-in zoom-in-95 duration-100"
            style={{top: y, left: x}}
        >
            {type === 'node' ? (
                <>
                    <button
                        onClick={() => handleAction('duplicate')}
                        className="w-full text-left px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center transition-colors"
                    >
                        <Copy size={14} className="mr-2 text-slate-400"/>
                        {t.contextMenu.duplicate}
                    </button>
                    <button
                        onClick={() => handleAction('copy_id')}
                        className="w-full text-left px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center transition-colors"
                    >
                        <Clipboard size={14} className="mr-2 text-slate-400"/>
                        {t.contextMenu.copyId}
                    </button>
                    <div className="h-[1px] bg-slate-100 dark:bg-slate-800 my-1"/>
                    <button
                        onClick={() => handleAction('delete')}
                        className="w-full text-left px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center transition-colors"
                    >
                        <Trash2 size={14} className="mr-2"/>
                        {t.common.delete}
                    </button>
                </>
            ) : (
                <>
                    <button
                        onClick={() => handleAction('fit_view')}
                        className="w-full text-left px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center transition-colors"
                    >
                        <Crosshair size={14} className="mr-2 text-slate-400"/>
                        {t.contextMenu.resetView}
                    </button>
                    {/* Future: Add 'Paste' or 'Add Note' here */}
                </>
            )}
        </div>
    );
};
