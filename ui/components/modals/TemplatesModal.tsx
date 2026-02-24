import React, {useState} from 'react';
import {FileText, LayoutTemplate, X} from 'lucide-react';
import {useUIStore} from '@/store/uiStore';
import {useDagStore} from '@/store/dagStore';
import {Button, ConfirmModal} from '../ui';
import {useTranslation} from '@/store/translations.ts';
import {TEMPLATES, createCategories, WorkflowTemplate} from '@/data/workflowTemplates';
import {TemplateCard} from '@/components/templates/TemplateCard';
import {CategoryTabs} from '@/components/templates/CategoryTabs';
import {useTemplateFilter} from '@/hooks/useTemplateFilter';

export const TemplatesModal: React.FC = () => {
    const {setActiveModal} = useUIStore();
    const {loadGraph} = useDagStore();
    const t = useTranslation();
    const [selectedCategory, setSelectedCategory] = useState<string>('all');
    const [searchQuery, setSearchQuery] = useState('');
    const [templateToLoad, setTemplateToLoad] = useState<WorkflowTemplate | null>(null);

    const categories = createCategories(t);
    const filteredTemplates = useTemplateFilter(TEMPLATES, selectedCategory, searchQuery);

    const handleSelect = (template: WorkflowTemplate) => {
        setTemplateToLoad(template);
    };

    const handleConfirmLoad = () => {
        if (templateToLoad) {
            const nodesCopy = JSON.parse(JSON.stringify(templateToLoad.nodes));
            const edgesCopy = JSON.parse(JSON.stringify(templateToLoad.edges));
            loadGraph(nodesCopy, edgesCopy);
            setActiveModal(null);
        }
        setTemplateToLoad(null);
    };

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div
                className="w-full max-w-5xl bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden transform animate-in zoom-in-95 duration-200 flex flex-col max-h-[90vh]">

                <div
                    className="p-6 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center bg-slate-50 dark:bg-slate-800/50">
                    <div>
                        <h2 className="text-xl font-bold text-slate-800 dark:text-slate-100 flex items-center">
                            <LayoutTemplate className="mr-3 text-blue-500" size={24}/>
                            {t.templates.title}
                        </h2>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                            {t.templates.subtitle}
                        </p>
                    </div>
                    <Button
                        onClick={() => setActiveModal(null)}
                        variant="ghost"
                        size="sm"
                        icon={<X size={20}/>}
                    />
                </div>

                <div
                    className="px-6 py-4 border-b border-slate-100 dark:border-slate-800 flex flex-wrap gap-3 items-center bg-white dark:bg-slate-900">
                    <CategoryTabs
                        categories={categories}
                        selectedCategory={selectedCategory}
                        onCategoryChange={setSelectedCategory}
                    />
                    <div className="flex-1 min-w-[200px]">
                        <input
                            type="text"
                            placeholder={t.templates.searchPlaceholder}
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full px-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-700 dark:text-slate-200 placeholder-slate-400"
                        />
                    </div>
                </div>

                <div className="flex-1 overflow-y-auto p-6 bg-slate-50/30 dark:bg-slate-950/30">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                        {filteredTemplates.map((template) => (
                            <TemplateCard
                                key={template.id}
                                template={template}
                                categories={categories}
                                onClick={handleSelect}
                            />
                        ))}
                    </div>

                    {filteredTemplates.length === 0 && (
                        <div className="text-center py-12 text-slate-400">
                            <FileText size={48} className="mx-auto mb-3 opacity-50"/>
                            <p>{t.templates.noResults}</p>
                        </div>
                    )}
                </div>

            </div>

            <ConfirmModal
                isOpen={!!templateToLoad}
                onClose={() => setTemplateToLoad(null)}
                onConfirm={handleConfirmLoad}
                title={t.templates.loadModal.title}
                message={t.templates.loadModal.message}
                confirmText={t.templates.loadModal.confirm}
                variant="warning"
            />
        </div>
    );
};
