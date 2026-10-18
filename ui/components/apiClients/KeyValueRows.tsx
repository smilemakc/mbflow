import React, {useEffect, useState} from 'react';
import {Plus, Trash2} from 'lucide-react';
import {VariableAutocomplete} from '@/components/builder/VariableAutocomplete';
import {configStyles} from '@/styles/configStyles';

interface Row {
    key: string;
    value: string;
}

interface KeyValueRowsProps {
    value: Record<string, string>;
    onChange: (value: Record<string, string>) => void;
    addLabel: string;
    keyPlaceholder?: string;
    valuePlaceholder?: string;
    enableVariables?: boolean;
    /** Rows are rebuilt from value only when this key changes (e.g. another node is selected). */
    resetKey?: string;
}

const toRows = (value: Record<string, string>): Row[] => Object.entries(value).map(([key, v]) => ({key, value: v}));

const toObject = (rows: Row[]): Record<string, string> => {
    const result: Record<string, string> = {};
    rows.forEach(({key, value}) => {
        if (key.trim()) result[key.trim()] = value;
    });
    return result;
};

/**
 * Editable list of key/value pairs. Rows live in local state, so a row
 * without a key yet does not disappear while the user types its value.
 */
export const KeyValueRows: React.FC<KeyValueRowsProps> = ({
    value,
    onChange,
    addLabel,
    keyPlaceholder,
    valuePlaceholder,
    enableVariables = false,
    resetKey,
}) => {
    const [rows, setRows] = useState<Row[]>(() => toRows(value));

    useEffect(() => {
        setRows(toRows(value));
    }, [resetKey]);

    const commit = (next: Row[]) => {
        setRows(next);
        onChange(toObject(next));
    };

    const update = (index: number, patch: Partial<Row>) =>
        commit(rows.map((row, i) => (i === index ? {...row, ...patch} : row)));

    return (
        <div className="space-y-2">
            {rows.map((row, index) => (
                <div key={index} className="grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)_auto] gap-1.5 items-center">
                    <input
                        type="text"
                        value={row.key}
                        onChange={(e) => update(index, {key: e.target.value})}
                        placeholder={keyPlaceholder}
                        className={`${configStyles.input} font-mono text-xs`}
                    />
                    {enableVariables ? (
                        <VariableAutocomplete
                            value={row.value}
                            onChange={(v) => update(index, {value: v})}
                            placeholder={valuePlaceholder}
                            className={`${configStyles.input} text-xs`}
                        />
                    ) : (
                        <input
                            type="text"
                            value={row.value}
                            onChange={(e) => update(index, {value: e.target.value})}
                            placeholder={valuePlaceholder}
                            className={`${configStyles.input} text-xs`}
                        />
                    )}
                    <button
                        type="button"
                        onClick={() => commit(rows.filter((_, i) => i !== index))}
                        className="p-1.5 rounded-md text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                        aria-label="Remove"
                    >
                        <Trash2 size={14}/>
                    </button>
                </div>
            ))}
            <button
                type="button"
                onClick={() => setRows([...rows, {key: '', value: ''}])}
                className="inline-flex items-center gap-1 text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline"
            >
                <Plus size={14}/>
                {addLabel}
            </button>
        </div>
    );
};
