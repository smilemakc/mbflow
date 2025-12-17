import {useMemo} from 'react';
import {WorkflowTemplate} from '@/data/workflowTemplates';

export function useTemplateFilter(
    templates: WorkflowTemplate[],
    category: string,
    searchQuery: string
) {
    return useMemo(() => {
        return templates.filter(template => {
            const matchesCategory = category === 'all' || template.category === category;
            const matchesSearch = searchQuery === '' ||
                template.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                template.description.toLowerCase().includes(searchQuery.toLowerCase());
            return matchesCategory && matchesSearch;
        });
    }, [templates, category, searchQuery]);
}
