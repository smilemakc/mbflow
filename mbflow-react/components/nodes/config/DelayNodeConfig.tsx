import React from 'react';
import {Clock} from 'lucide-react';
import type {DelayNodeConfig as DelayNodeConfigType} from '@/types/nodeConfigs.ts';
import {useTranslation} from '@/store/translations';

interface DelayNodeConfigProps {
    config: DelayNodeConfigType;
    nodeId?: string;
    onChange: (config: DelayNodeConfigType) => void;
}

export const DelayNodeConfig: React.FC<DelayNodeConfigProps> = ({
                                                                    config,
                                                                    onChange,
                                                                }) => {
    const t = useTranslation();
    const handleDurationChange = (duration: number) => {
        onChange({
            ...config,
            duration: Math.max(0, duration),
        });
    };

    const handleUnitChange = (unit: 'seconds' | 'minutes' | 'hours') => {
        onChange({
            ...config,
            unit,
        });
    };

    const handleDescriptionChange = (description: string) => {
        onChange({
            ...config,
            description,
        });
    };

    return (
        <div className="space-y-6">
            <div
                className="bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-900/10 dark:to-blue-900/10 border border-cyan-200 dark:border-cyan-800 rounded-lg p-4 flex items-start gap-3">
                <Clock className="text-cyan-600 dark:text-cyan-400 flex-shrink-0 mt-0.5" size={18}/>
                <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white text-sm">{t.nodeConfig.delay.title}</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
                        {t.nodeConfig.delay.description}
                    </p>
                </div>
            </div>

            {/* Duration Section */}
            <div className="space-y-3">
                <label className="block">
          <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
            {t.nodeConfig.delay.duration}
          </span>
                    <div className="flex gap-2">
                        <input
                            type="number"
                            min="0"
                            value={config.duration || 0}
                            onChange={(e) => handleDurationChange(parseInt(e.target.value, 10) || 0)}
                            className="flex-1 px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 dark:focus:ring-cyan-400 text-sm"
                            placeholder={t.nodeConfig.delay.durationPlaceholder}
                        />
                        <select
                            value={config.unit || 'seconds'}
                            onChange={(e) => handleUnitChange(e.target.value as 'seconds' | 'minutes' | 'hours')}
                            className="px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 dark:focus:ring-cyan-400 text-sm font-medium"
                        >
                            <option value="seconds">{t.nodeConfig.delay.unitSeconds}</option>
                            <option value="minutes">{t.nodeConfig.delay.unitMinutes}</option>
                            <option value="hours">{t.nodeConfig.delay.unitHours}</option>
                        </select>
                    </div>
                </label>
            </div>

            {/* Description Section */}
            <div className="space-y-3">
                <label className="block">
          <span className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block">
            {t.nodeConfig.delay.descriptionLabel} <span className="font-normal text-slate-500 dark:text-slate-400">{t.nodeConfig.delay.descriptionOptional}</span>
          </span>
                    <textarea
                        value={config.description || ''}
                        onChange={(e) => handleDescriptionChange(e.target.value)}
                        rows={3}
                        className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 dark:focus:ring-cyan-400 text-sm resize-none"
                        placeholder={t.nodeConfig.delay.descriptionPlaceholder}
                    />
                </label>
            </div>

            {/* Preview */}
            <div
                className="bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-800 rounded-lg p-3">
                <p className="text-xs text-slate-600 dark:text-slate-400 font-medium mb-2">{t.nodeConfig.delay.preview}</p>
                <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300 font-mono">
                    <Clock size={16} className="text-slate-400"/>
                    <span>
            {t.nodeConfig.delay.previewWait}{' '}
                        <span className="font-semibold text-slate-900 dark:text-white">
              {config.duration || 0} {config.unit || 'seconds'}
            </span>
          </span>
                </div>
            </div>
        </div>
    );
};
