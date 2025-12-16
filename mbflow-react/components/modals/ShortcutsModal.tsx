import React from 'react';
import {Keyboard, X} from 'lucide-react';
import {useUIStore} from '@/store/uiStore';
import { Button } from '../ui';
import { useTranslation } from '@/store/translations';

export const ShortcutsModal: React.FC = () => {
    const {setActiveModal} = useUIStore();
    const t = useTranslation();

    const SHORTCUTS = [
        {key: 'Ctrl + S', description: t.shortcuts.saveWorkflow},
        {key: 'Ctrl + Enter', description: t.shortcuts.runWorkflow},
        {key: 'Ctrl + Z', description: t.shortcuts.undo},
        {key: 'Ctrl + Y', description: t.shortcuts.redo},
        {key: 'Shift + F', description: t.shortcuts.toggleFocusMode},
        {key: 'Delete', description: t.shortcuts.deleteNode},
        {key: 'Ctrl + /', description: t.shortcuts.toggleShortcuts},
    ];

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div
                className="w-full max-w-md bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden transform animate-in zoom-in-95 duration-200">
                <div
                    className="p-4 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center bg-slate-50 dark:bg-slate-800/50">
                    <div className="flex items-center space-x-2">
                        <Keyboard className="text-blue-500" size={20}/>
                        <h2 className="font-bold text-slate-800 dark:text-slate-100">{t.shortcuts.title}</h2>
                    </div>
                    <Button
                        onClick={() => setActiveModal(null)}
                        variant="ghost"
                        size="sm"
                        icon={<X size={18} />}
                    />
                </div>

                <div className="p-2">
                    <div className="space-y-1">
                        {SHORTCUTS.map((item, idx) => (
                            <div key={idx}
                                 className="flex justify-between items-center px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800 rounded-lg transition-colors group">
                <span
                    className="text-sm font-medium text-slate-600 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white transition-colors">
                  {item.description}
                </span>
                                <span className="flex items-center space-x-1">
                  {item.key.split(' ').map((k, i) => (
                      <kbd key={i}
                           className="px-2 py-1 bg-slate-100 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 rounded-md text-xs font-mono font-bold text-slate-600 dark:text-slate-400 min-w-[24px] text-center shadow-sm">
                          {k === 'Ctrl' ? (navigator.platform.includes('Mac') ? '⌘' : 'Ctrl') : k}
                      </kbd>
                  ))}
                </span>
                            </div>
                        ))}
                    </div>
                </div>

                <div
                    className="p-4 bg-slate-50 dark:bg-slate-800/50 border-t border-slate-100 dark:border-slate-800 text-center">
                    <p className="text-xs text-slate-400">{t.shortcuts.proTip}</p>
                </div>
            </div>
        </div>
    );
};
