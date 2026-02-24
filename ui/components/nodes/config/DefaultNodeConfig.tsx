import React from 'react';
import { NodeConfigProps } from './nodeConfigRegistry';
import { useTranslation } from '@/store/translations';

export const DefaultNodeConfig: React.FC<NodeConfigProps> = () => {
    const t = useTranslation();

    return (
        <div className="text-sm text-slate-400 dark:text-slate-500 italic">
            {t.builder.noConfigAvailable || 'No specific configuration available for this node type.'}
        </div>
    );
};
