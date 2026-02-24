import React from 'react';
import {ArrowRight, Info} from 'lucide-react';
import { useTranslation } from '@/store/translations.ts';

interface AdapterConfigProps {
    config: Record<string, any>;
    nodeId?: string;
    onChange: (config: Record<string, any>) => void;
}

const InfoBlock: React.FC<{ input: string; output: string }> = ({input, output}) => (
    <div
        className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg">
        <Info className="w-4 h-4 flex-shrink-0 text-blue-500"/>
        <div className="flex items-center gap-2 flex-1">
            <code className="bg-slate-200 dark:bg-slate-800 px-2 py-1 rounded font-mono text-[11px]">
                {input}
            </code>
            <ArrowRight className="w-3 h-3 flex-shrink-0"/>
            <code className="bg-slate-200 dark:bg-slate-800 px-2 py-1 rounded font-mono text-[11px]">
                {output}
            </code>
        </div>
    </div>
);

export const Base64ToBytesNodeConfig: React.FC<AdapterConfigProps> = () => {
    const t = useTranslation();
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.base64ToBytes.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.base64ToBytes.description}
                </p>
            </div>

            <InfoBlock
                input="SGVsbG8gV29ybGQ="
                output="b'Hello World'"
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                {t.nodeConfig.adapter.base64ToBytes.noConfig}
            </div>
        </div>
    );
};

export const BytesToBase64NodeConfig: React.FC<AdapterConfigProps> = () => {
    const t = useTranslation();
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.bytesToBase64.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.bytesToBase64.description}
                </p>
            </div>

            <InfoBlock
                input="b'Hello World'"
                output="SGVsbG8gV29ybGQ="
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                {t.nodeConfig.adapter.bytesToBase64.noConfig}
            </div>
        </div>
    );
};

export const StringToJsonNodeConfig: React.FC<AdapterConfigProps> = () => {
    const t = useTranslation();
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.stringToJson.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.stringToJson.description}
                </p>
            </div>

            <InfoBlock
                input='"{\"name\":\"Alice\"}"'
                output="{name: 'Alice'}"
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                {t.nodeConfig.adapter.stringToJson.noConfig}
            </div>
        </div>
    );
};

export const JsonToStringNodeConfig: React.FC<AdapterConfigProps> = () => {
    const t = useTranslation();
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.jsonToString.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.jsonToString.description}
                </p>
            </div>

            <InfoBlock
                input="{name: 'Alice'}"
                output='"{\"name\":\"Alice\"}"'
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                {t.nodeConfig.adapter.jsonToString.noConfig}
            </div>
        </div>
    );
};

export const BytesToJsonNodeConfig: React.FC<AdapterConfigProps> = () => {
    const t = useTranslation();
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.bytesToJson.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.bytesToJson.description}
                </p>
            </div>

            <InfoBlock
                input={'b\'{"name":"Alice"}\''}
                output={'{"name": "Alice"}'}
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                {t.nodeConfig.adapter.bytesToJson.noConfig}
            </div>
        </div>
    );
};

export const FileToBytesNodeConfig: React.FC<AdapterConfigProps> = () => {
    const t = useTranslation();
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.fileToBytes.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.fileToBytes.description}
                </p>
            </div>

            <InfoBlock
                input="File(id='abc123')"
                output="b'file content...'"
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                {t.nodeConfig.adapter.fileToBytes.noConfig}
            </div>
        </div>
    );
};

export const BytesToFileNodeConfig: React.FC<AdapterConfigProps> = ({config, onChange}) => {
    const t = useTranslation();
    const handleFilenameChange = (filename: string) => {
        onChange({...config, filename});
    };

    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    {t.nodeConfig.adapter.bytesToFile.title}
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    {t.nodeConfig.adapter.bytesToFile.description}
                </p>
            </div>

            <InfoBlock
                input="b'content...'"
                output="File(id='xyz789')"
            />

            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    {t.nodeConfig.adapter.bytesToFile.filename}
                </label>
                <input
                    type="text"
                    value={config.filename || ''}
                    onChange={(e) => handleFilenameChange(e.target.value)}
                    placeholder={t.nodeConfig.adapter.bytesToFile.filenamePlaceholder}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200 placeholder-slate-400"
                />
                <p className="text-xs text-slate-500 dark:text-slate-500">
                    {t.nodeConfig.adapter.bytesToFile.filenameHint}
                </p>
            </div>
        </div>
    );
};
