import React, {useEffect, useState} from 'react';
import {Plus, Trash2} from 'lucide-react';
import {VariableAutocomplete} from '@/components/builder/VariableAutocomplete';
import {Button} from '../ui';

interface KeyValueItem {
    key: string;
    value: string;
}

interface Props {
    value: Record<string, string>;
    onChange: (value: Record<string, string>) => void;
    nodeId?: string;
    placeholderKey?: string;
    placeholderValue?: string;
    itemLabel?: string;
}

export const KeyValueEditor: React.FC<Props> = ({
                                                    value,
                                                    onChange,
                                                    nodeId,
                                                    placeholderKey = 'Key',
                                                    placeholderValue = 'Value',
                                                    itemLabel = 'Item',
                                                }) => {
    const [items, setItems] = useState<KeyValueItem[]>([]);

    const objectToItems = (obj: Record<string, string>): KeyValueItem[] => {
        return Object.entries(obj).map(([key, value]) => ({key, value}));
    };

    const itemsToObject = (items: KeyValueItem[]): Record<string, string> => {
        const result: Record<string, string> = {};
        items.forEach(({key, value}) => {
            if (key.trim()) {
                result[key] = value;
            }
        });
        return result;
    };

    useEffect(() => {
        const newItems = objectToItems(value);
        if (newItems.length === 0) {
            setItems([{key: '', value: ''}]);
        } else {
            setItems(newItems);
        }
    }, [value]);

    const updateKey = (index: number, newKey: string) => {
        const newItems = [...items];
        if (newItems[index]) {
            newItems[index].key = newKey;
            setItems(newItems);
            onChange(itemsToObject(newItems));
        }
    };

    const updateValue = (index: number, newValue: string) => {
        const newItems = [...items];
        if (newItems[index]) {
            newItems[index].value = newValue;
            setItems(newItems);
            onChange(itemsToObject(newItems));
        }
    };

    const addItem = () => {
        setItems([...items, {key: '', value: ''}]);
    };

    const removeItem = (index: number) => {
        const newItems = items.filter((_, i) => i !== index);
        if (newItems.length === 0) {
            newItems.push({key: '', value: ''});
        }
        setItems(newItems);
        onChange(itemsToObject(newItems));
    };

    return (
        <div className="flex flex-col gap-2">
            {items.map((item, index) => (
                <div key={index} className="grid grid-cols-[1fr_2fr_auto] gap-2 items-start">
                    <input
                        type="text"
                        value={item.key}
                        onChange={(e) => updateKey(index, e.target.value)}
                        placeholder={placeholderKey}
                        className="min-w-0 px-3 py-2 text-sm bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent text-slate-900 dark:text-slate-100 placeholder-slate-400 dark:placeholder-slate-500"
                    />
                    <VariableAutocomplete
                        value={item.value}
                        onChange={(newValue) => updateValue(index, newValue)}
                        placeholder={placeholderValue}
                        className="min-w-0 px-3 py-2 text-sm bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent text-slate-900 dark:text-slate-100 placeholder-slate-400 dark:placeholder-slate-500"
                    />
                    <Button
                        type="button"
                        onClick={() => removeItem(index)}
                        variant="danger"
                        size="sm"
                        icon={<Trash2 size={16}/>}
                        title="Remove"
                    />
                </div>
            ))}

            <Button
                type="button"
                onClick={addItem}
                variant="outline"
                size="sm"
                icon={<Plus size={16}/>}
            >
                Add {itemLabel}
            </Button>
        </div>
    );
};
