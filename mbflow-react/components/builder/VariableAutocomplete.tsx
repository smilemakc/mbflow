import React, {useRef, useState} from 'react';
import {useDagStore} from '@/store/dagStore.ts';
import {Variable, VariableSource, VariableType} from '@/types.ts';
import {Box, Braces, CheckSquare, Globe, Hash, Layers, Type} from 'lucide-react';

interface Props {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    className?: string;
    type?: 'input' | 'textarea';
    rows?: number;
}

const VariableIcon: React.FC<{ type: VariableType; source: VariableSource }> = ({type, source}) => {
    let Icon = Braces;
    let colorClass = 'text-slate-500 dark:text-slate-400';

    if (source === VariableSource.GLOBAL) {
        Icon = Globe;
        colorClass = 'text-purple-500 dark:text-purple-400';
    } else if (source === VariableSource.NODE) {
        Icon = Layers;
        colorClass = 'text-blue-500 dark:text-blue-400';
    } else {
        switch (type) {
            case VariableType.STRING:
                Icon = Type;
                colorClass = 'text-slate-500 dark:text-slate-400';
                break;
            case VariableType.NUMBER:
                Icon = Hash;
                colorClass = 'text-green-500 dark:text-green-400';
                break;
            case VariableType.BOOLEAN:
                Icon = CheckSquare;
                colorClass = 'text-orange-500 dark:text-orange-400';
                break;
            case VariableType.OBJECT:
                Icon = Box;
                colorClass = 'text-indigo-500 dark:text-indigo-400';
                break;
        }
    }

    return <Icon size={14} className={colorClass}/>;
};

export const VariableAutocomplete: React.FC<Props> = (
    {
        value = '',
        onChange,
        placeholder,
        className,
        type = 'input',
        rows
    }) => {
    const {getAvailableVariables} = useDagStore();
    const [isOpen, setIsOpen] = useState(false);
    const [query, setQuery] = useState('');
    const [cursorPos, setCursorPos] = useState(0);
    const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

    // Get variables from store
    const variables = getAvailableVariables();

    // Filter based on query after {{
    const filteredVars = variables.filter(v =>
        v.key.toLowerCase().includes(query.toLowerCase()) ||
        v.name.toLowerCase().includes(query.toLowerCase())
    );

    const handleInput = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const newVal = e.target.value;
        const newPos = e.target.selectionStart || 0;

        onChange(newVal);
        setCursorPos(newPos);

        // Check for trigger '{{' before cursor
        const textBefore = newVal.slice(0, newPos);
        // Regex matches {{ followed by any word chars until end of string (cursor)
        const match = textBefore.match(/\{\{([a-zA-Z0-9_\.]*)$/);

        if (match) {
            setIsOpen(true);
            setQuery(match[1]);
        } else {
            setIsOpen(false);
        }
    };

    const handleSelect = (variable: Variable) => {
        const textBefore = value.slice(0, cursorPos);
        const textAfter = value.slice(cursorPos);

        // Find the trigger again to know where to replace
        const match = textBefore.match(/\{\{([a-zA-Z0-9_\.]*)$/);

        if (match && match.index !== undefined) {
            const prefix = textBefore.slice(0, match.index);
            const inserted = `{{${variable.key}}}`;
            const newValue = `${prefix}${inserted}${textAfter}`;

            onChange(newValue);
            setIsOpen(false);

            // Restore focus and move cursor after the inserted variable
            setTimeout(() => {
                if (inputRef.current) {
                    const newCursorPos = prefix.length + inserted.length;
                    inputRef.current.focus();
                    inputRef.current.setSelectionRange(newCursorPos, newCursorPos);
                }
            }, 0);
        }
    };

    const handleBlur = () => {
        // Delay hide to allow click events on dropdown items to register
        setTimeout(() => setIsOpen(false), 200);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (isOpen) {
            if (e.key === 'Escape') {
                setIsOpen(false);
                e.preventDefault();
            }
            // Future: Add arrow key navigation
        }
    };

    return (
        <div className="relative w-full">
            {type === 'textarea' ? (
                <textarea
                    ref={inputRef as any}
                    value={value}
                    onChange={handleInput}
                    onBlur={handleBlur}
                    onKeyDown={handleKeyDown}
                    placeholder={placeholder}
                    rows={rows || 3}
                    className={className}
                />
            ) : (
                <input
                    ref={inputRef as any}
                    type="text"
                    value={value}
                    onChange={handleInput}
                    onBlur={handleBlur}
                    onKeyDown={handleKeyDown}
                    placeholder={placeholder}
                    className={className}
                />
            )}

            {isOpen && filteredVars.length > 0 && (
                <div
                    className="absolute left-0 right-0 top-full mt-1 max-h-48 overflow-y-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg shadow-xl shadow-slate-200/50 dark:shadow-black/50 z-50">
                    <div className="py-1">
                        <div
                            className="px-3 py-1.5 text-[10px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider bg-slate-50 dark:bg-slate-950 border-b border-slate-100 dark:border-slate-800">
                            Suggested Variables
                        </div>
                        {filteredVars.map((v) => (
                            <button
                                key={v.id}
                                onMouseDown={(e) => {
                                    e.preventDefault(); // Prevent blur on input
                                    handleSelect(v);
                                }}
                                className="w-full text-left px-3 py-2 text-sm hover:bg-blue-50 dark:hover:bg-blue-900/20 hover:text-blue-700 dark:hover:text-blue-400 flex items-center group transition-colors text-slate-700 dark:text-slate-200"
                            >
                                <div
                                    className="flex-shrink-0 mr-2.5 p-1 bg-slate-100 dark:bg-slate-800 rounded group-hover:bg-blue-100 dark:group-hover:bg-blue-900/50 transition-colors">
                                    <VariableIcon type={v.type} source={v.source}/>
                                </div>
                                <div className="flex-1 min-w-0">
                                    <div className="font-medium truncate">{v.name}</div>
                                    <div
                                        className="text-xs text-slate-400 dark:text-slate-500 font-mono truncate">{`{{${v.key}}}`}</div>
                                </div>
                            </button>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
};