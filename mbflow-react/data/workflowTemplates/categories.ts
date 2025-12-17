export interface CategoryOption {
    id: string;
    label: string;
}

export const createCategories = (t: any): CategoryOption[] => [
    {id: 'all', label: t.templates.allTemplates},
    {id: 'basic', label: t.templates.basic},
    {id: 'telegram', label: t.templates.telegram},
    {id: 'ai', label: t.templates.aiLlm},
    {id: 'data', label: t.templates.dataProcessing}
];
