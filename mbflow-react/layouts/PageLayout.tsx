import React from 'react';
import { useUIStore } from '@/store/uiStore';
import { Sidebar } from '@/components/navigation/Sidebar';
import { ShortcutsModal } from '@/components/modals/ShortcutsModal';
import { TemplatesModal } from '@/components/modals/TemplatesModal';

interface PageLayoutProps {
  children: React.ReactNode;
  title?: string;
}

export const PageLayout: React.FC<PageLayoutProps> = ({ children, title }) => {
  const { activeModal } = useUIStore();

  return (
    <div className="flex h-screen w-screen bg-slate-50 dark:bg-slate-950 overflow-hidden transition-colors">
      {/* Modals */}
      {activeModal === 'templates' && <TemplatesModal />}
      {activeModal === 'shortcuts' && <ShortcutsModal />}

      <Sidebar />

      <div className="flex-1 flex flex-col h-full overflow-hidden">
        {/* Page Header */}
        <div className="h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center px-6 shrink-0 z-10">
          <h1 className="text-xl font-bold text-slate-800 dark:text-white">
            {title || 'MBFlow'}
          </h1>
        </div>
        {children}
      </div>
    </div>
  );
};
