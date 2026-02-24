import React from 'react';
import {CategoryOption} from '@/data/workflowTemplates';

interface CategoryTabsProps {
    categories: CategoryOption[];
    selectedCategory: string;
    onCategoryChange: (categoryId: string) => void;
}

export const CategoryTabs: React.FC<CategoryTabsProps> = ({
                                                               categories,
                                                               selectedCategory,
                                                               onCategoryChange
                                                           }) => {
    return (
        <div className="flex gap-2 flex-wrap">
            {categories.map(cat => (
                <button
                    key={cat.id}
                    onClick={() => onCategoryChange(cat.id)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                        selectedCategory === cat.id
                            ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`}
                >
                    {cat.label}
                </button>
            ))}
        </div>
    );
};
