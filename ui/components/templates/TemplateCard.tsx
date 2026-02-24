import React from 'react';
import {ArrowRight} from 'lucide-react';
import {WorkflowTemplate, CategoryOption} from '@/data/workflowTemplates';
import {useTranslation} from '@/store/translations';

interface TemplateCardProps {
    template: WorkflowTemplate;
    categories: CategoryOption[];
    onClick: (template: WorkflowTemplate) => void;
}

export const TemplateCard: React.FC<TemplateCardProps> = ({template, categories, onClick}) => {
    const t = useTranslation();

    return (
        <div
            className="group bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 hover:border-blue-400 dark:hover:border-blue-600 hover:shadow-lg hover:shadow-blue-500/10 transition-all cursor-pointer flex flex-col"
            onClick={() => onClick(template)}
        >
            <div className="flex justify-between items-start mb-3">
                <div
                    className={`p-2.5 rounded-lg bg-${template.color}-50 dark:bg-${template.color}-900/20 text-${template.color}-600 dark:text-${template.color}-400 group-hover:scale-110 transition-transform`}>
                    <template.icon size={22}/>
                </div>
                <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                    <span
                        className="flex items-center text-xs font-bold text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30 px-2 py-1 rounded-full">
                        {t.templates.use} <ArrowRight size={12} className="ml-1"/>
                    </span>
                </div>
            </div>

            <h3 className="font-bold text-lg text-slate-900 dark:text-white mb-1.5">{template.name}</h3>
            <p className="text-sm text-slate-500 dark:text-slate-400 mb-4 flex-1 leading-relaxed">{template.description}</p>

            <div className="flex items-center justify-between pt-3 border-t border-slate-100 dark:border-slate-800">
                <div className="flex items-center space-x-3">
                    <span
                        className="text-xs font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                        {template.nodes.length} {t.templates.nodesCount}
                    </span>
                    <span
                        className="text-xs font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                        {template.edges.length} {t.templates.edgesCount}
                    </span>
                </div>
                <span
                    className={`text-xs font-medium px-2 py-0.5 rounded-full bg-${template.color}-50 dark:bg-${template.color}-900/20 text-${template.color}-600 dark:text-${template.color}-400`}>
                    {categories.find(c => c.id === template.category)?.label}
                </span>
            </div>
        </div>
    );
};
